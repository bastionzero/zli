import { cleanExit } from '../../handlers/clean-exit.handler';
import { SSHConfigHostBlock, ValidSSHHost, SSHHostConfig, SSHConfigParseError, InvalidSSHHost, ValidSSHHostAndConfig, QuickstartSSMTarget, RegistrableSSHHost } from './quickstart-ssm.service.types';
import { getAutodiscoveryScript } from '../auto-discovery-script/auto-discovery-script.service';
import { ConfigService } from '../config/config.service';
import { Logger } from '../logger/logger.service';
import { SsmTargetService } from '../ssm-target/ssm-target.service';
import { TargetStatus } from '../common.types';
import { readFile } from '../../utils';

import SSHConnection from 'ssh2-promise/lib/sshConnection';
import path from 'path';
import os from 'os';
import prompts, { PromptObject } from 'prompts';
import { KeyEncryptedError, parsePrivateKey } from 'sshpk';
import { PolicyService } from '../policy/policy.service';
import { PolicyEnvironment, PolicySummary, PolicyTargetUser, PolicyType, Subject, SubjectType, TargetConnectContext } from '../policy/policy.types';
import { Verb, VerbType } from '../policy-query/policy-query.types';
import { EnvironmentService } from '../environment/environment.service';
import { SsmTargetSummary } from '../ssm-target/ssm-target.types';
import { Retrier } from '@jsier/retrier';

export class QuickstartSsmService {
    constructor(
        private logger: Logger,
        private configService: ConfigService,
        private policyService: PolicyService,
        private environmentService: EnvironmentService
    ) { }

    /**
     * Polls the bastion until the SSM target is Online and the agent version is
     * known.
     * @param ssmTargetId The ID of the target to poll
     * @returns Information about the target
     */
    public async pollSsmTargetOnline(ssmTargetId: string) : Promise<SsmTargetSummary> {
        // Try 30 times with a delay of 10 seconds between each attempt.
        const retrier = new Retrier({
            limit: 30,
            delay: 1000 * 10
        });
        return retrier.resolve(() => new Promise<SsmTargetSummary>(async (resolve, reject) => {
            try {
                const ssmTargetService = new SsmTargetService(this.configService, this.logger);

                const target = await ssmTargetService.GetSsmTarget(ssmTargetId);
                if (target.status === TargetStatus.Online && target.agentVersion !== '') {
                    resolve(target);
                } else {
                    this.logger.info(`Waiting for ${target.name} to come online...`);
                    this.logger.debug(`Target ${target.name} has status:${target.status.toString()} and agentVersion:${target.agentVersion}`);
                    reject(`Target ${target.name} is not online`);
                }
            } catch (error) {
                reject(error);
            }
        }));
    }

    /**
     * Check to see if an agent is already installed on a host
     * @param sshConnection SSH connection to the host to be checked
     * @param hostName Name of the host
     * @returns True if the agent is already installed. False otherwise.
     */
    private async isAgentAlreadyInstalled(sshConnection: SSHConnection, hostName: string): Promise<boolean> {
        try {
            // Check to see if agent is already installed on this host
            //
            // NOTE: We don't handle the edge case where the executable name
            // has changed since the target was first registered. In this
            // edge case, the target will be registered again.
            await sshConnection.exec(`bzero-ssm-agent --version`);
        } catch {
            // exec() throws an error if the command fails to run (e.g. agent
            // binary not found)
            this.logger.debug(`Agent not found on host ${hostName}`);
            return false;
        }

        // If the catch block wasn't hit, then we know the agent is installed as
        // the command succeeded.
        return true;
    }

    /**
     * Attempts to add a registrable SSH host to BastionZero. Fails with a
     * rejected promise if any step of the registration process fails: getting
     * autodiscovery script from Bastion, running autodiscovery script, and
     * polling for target to be online.
     *
     * Note: This function fails if it is found that the passed in SSH host has
     * an agent already installed on it.
     * @param registrableHost The SSH host to register
     * @returns Target summary of the newly registered host
     */
    public async addSSHHostToBastionZero(registrableHost: RegistrableSSHHost): Promise<SsmTargetSummary> {
        return new Promise<SsmTargetSummary>(async (resolve, reject) => {
            try {
                this.logger.info(`Attempting to add SSH host ${registrableHost.host.sshHost.name} to BastionZero...`);

                this.logger.info(`Running autodiscovery script on SSH host ${registrableHost.host.sshHost.name} (could take several minutes)...`);
                const ssmTargetId = await this.runAutodiscoveryOnSSHHost(registrableHost);

                this.logger.info(`Bastion assigned SSH host ${registrableHost.host.sshHost.name} with the following unique target id: ${ssmTargetId}`);

                // Poll for "Online" status
                this.logger.info(`Waiting for target ${registrableHost.host.sshHost.name} to become online (could take several minutes)...`);
                const ssmTarget = await this.pollSsmTargetOnline(ssmTargetId);
                this.logger.info(`SSH host ${registrableHost.host.sshHost.name} successfully added to BastionZero!`);

                resolve(ssmTarget);
            } catch (error) {
                this.logger.error(`Failed to add SSH host: ${registrableHost.host.sshHost.name} to BastionZero. ${error}`);
                reject(error);
            }
        });
    }

    /**
     * Connects to an SSH host and runs the universal autodiscovery script on
     * it. This function returns a rejected promise if:
     * - The SSH connection failed
     * - The agent is already installed on the machine
     * - A failure to receive the autodiscovery script from the Bastion
     * - A failure to start the autodiscovery script
     * - If the autodiscovery script returns a non-zero exit status code
     * - A failure to parse the machine's SSM target ID
     * @param registrableSSHHost The SSH host to run the autodiscovery script on
     * @returns The target ID of the newly registered host
     */
    private async runAutodiscoveryOnSSHHost(registrableSSHHost: RegistrableSSHHost): Promise<string> {
        const sshConfig = registrableSSHHost.host.config;
        const hostName = registrableSSHHost.host.sshHost.name;

        // Timeout the ssh connection handshake after 20s and dont retry
        sshConfig.readyTimeout = 20 * 1000;
        sshConfig.reconnect = false;

        // Start SSH connection
        const ssh = new SSHConnection(sshConfig);
        let conn: SSHConnection;
        try {
            conn = await ssh.connect(sshConfig);
            this.logger.debug(`SSH connection established with host: ${hostName}`);
        }
        catch (error) {
            throw new Error(`Failed to establish SSH connection: ${error}`);
        }

        // Wrap everything in a try+finally block so that we close the SSH
        // connection on any kind of failure
        try {
            // Check to see if the host has already been registered by checking
            // to see if the agent is installed
            if (await this.isAgentAlreadyInstalled(conn, hostName)) {
                // We don't want to register twice, so fail early
                //
                // NOTE: Instead of throwing an error, a better design would be
                // to return the SSM target ID of the already registered target.
                // It improves the quickstart experience in the following ways:
                // (1) User's SSH config for an already registered SSH host can
                // have a new username => New policy will be created with new
                // TargetUser.
                // (2) If user picked this host as the target to connect to in
                // the end, it will still work.
                // (3) The target is still displayed in final summary that is
                // printed at the end of quickstart.
                //
                // We can't make this improvement right now as the agent doesn't
                // know its own ID (we don't store it in the agent)
                throw new Error('Agent is already installed');
            }

            // Get autodiscovery script
            //
            // The registered target's name will match the hostname alias parsed
            // from the SSH config file
            const script = await getAutodiscoveryScript(this.logger, this.configService, registrableSSHHost.envId, { scheme: 'manual', name: hostName }, 'universal', 'latest');

            // Run script on target
            const execAutodiscoveryScriptCmd = `bash << 'endmsg'\n${script}\nendmsg`;
            const execAutodiscoveryScript = new Promise<string>(async (resolve, reject) => {
                conn.spawn(execAutodiscoveryScriptCmd)
                    .then(socket => {
                        this.logger.debug(`Running autodiscovery script on host: ${hostName}`);

                        // Store last printed message on stdout
                        let lastOutput = '';

                        socket.on('data', (data: Buffer) => {
                            // Log stdout
                            const dataAsStr = data.toString();
                            this.logger.debug(`STDOUT: ${dataAsStr}`);
                            lastOutput = dataAsStr;
                        });
                        socket.on('close', (code: number) => {
                            if (code == 0) {
                                this.logger.debug(`Successfully executed autodiscovery script on host: ${hostName}`);

                                // Regex expression looks for "Target Id:" and then
                                // captures (saves a backwards reference) any
                                // alphanumeric (+ underscore) characters
                                const targetIdRegex = new RegExp('Target Id: ([\\w-]*)');
                                const matches = targetIdRegex.exec(lastOutput);
                                if (matches) {
                                    // The result of the first capturing group is
                                    // stored at index 1
                                    resolve(matches[1].trim());
                                } else {
                                    reject(`Failed to find target ID in last message printed by stdout`);
                                }
                            } else {
                                reject(`Failed to execute autodiscovery script. Error code: ${code}`);
                            }
                        });
                    })
                    .catch(err => {
                        reject(`Error when attempting to execute autodiscovery script on host: ${hostName}. ${err}`);
                    });
            });

            // Wait for the script to finish executing
            return await execAutodiscoveryScript;
        } finally {
            this.logger.debug(`Closing SSH connection with host: ${hostName}`);
            await conn.close();
            this.logger.debug(`Closed SSH connection with host: ${hostName}`);
        }
    }

    /**
     * Display a prompt asking the user if they wish to skip the host or exit
     * out of quickstart
     * @param hostName Name of the host
     * @param onCancel Handler function for interruption of the prompt (e.g.
     * CTRL-C, ESC, etc.)
     * @returns True if the user wishes to skip the host. False if the user
     * wishes to exit.
     */
    public async promptSkipHostOrExit(hostName: string, onCancel: (prompt: PromptObject, answers: any) => void): Promise<boolean> {
        const confirmSkipOrExit = await prompts({
            type: 'toggle',
            name: 'value',
            message: `Do you want to skip host ${hostName} or exit?`,
            initial: true,
            active: 'skip',
            inactive: 'exit',
        }, { onCancel: onCancel });

        return confirmSkipOrExit.value;
    }

    /**
     * Converts parsed valid SSH hosts from the SSH config file to SSHConfigs
     * that are usable by the ssh2-promise library. This process is interactive
     * as it also checks to see if the IdentityFile (SSH key) is both present on
     * disk and encrypted. If the key is encrypted, we prompt the user to enter
     * the passphrase so that we can decrypt it and use when building the SSH
     * connection later.
     * @param hosts List of valid SSH hosts
     * @param onCancel Handler function for interruption of the prompt (e.g.
     * CTRL-C, ESC, etc.)
     * @returns A list of valid SSH hosts with their corresponding SSHConfig
     * configurations for use with the ssh2-promise library
     */
    public async promptConvertValidSSHHostsToSSHConfigs(hosts: ValidSSHHost[], onCancel: (prompt: PromptObject, answers: any) => void): Promise<ValidSSHHostAndConfig[]> {
        const sshConfigs: ValidSSHHostAndConfig[] = [];
        for (const host of hosts) {
            this.logger.info(`Validating SSH key for host ${host.name}`);
            // Try to read the IdentityFile and store its contents in keyFile variable
            let keyFile: string;
            try {
                keyFile = await readFile(host.identityFile);
            } catch (err) {
                this.logger.error(`Removing ${host.name} from list of SSH hosts to add to BastionZero due to error when reading IdentityFile: ${err}`);
                continue;
            }

            // Check if IdentityFile is encrypted.
            let passphraseKeyFile: string;
            try {
                parsePrivateKey(keyFile, 'auto');
            } catch (err) {
                if (err instanceof KeyEncryptedError) {
                    this.logger.info(`${host.name}'s IdentityFile (${host.identityFile}) is encrypted!`);

                    // Ask user for password to decrypt the key file
                    const passwordResponse = await this.handleEncryptedIdentityFile(host.identityFile, keyFile);

                    // Check if user wants to skip this host or exit immediately
                    if (passwordResponse === undefined) {
                        const shouldSkip = await this.promptSkipHostOrExit(host.name, onCancel);

                        if (shouldSkip) {
                            this.logger.warn(`Removing ${host.name} from list of SSH hosts to add to BastionZero due to user not providing password to encrypted IdentityFile`);
                            continue;
                        } else {
                            this.logger.info('Prompt cancelled. Exiting out of quickstart...');
                            await cleanExit(1, this.logger);
                        }
                    } else {
                        passphraseKeyFile = passwordResponse;
                    }
                } else {
                    this.logger.error(`Removing ${host.name} from list of SSH hosts to add to BastionZero due to error when parsing IdentityFile: ${err}`);
                    continue;
                }
            }

            // Convert from ValidSSHHost to ValidSSHConfig
            this.logger.info(`SSH key for host ${host.name} is valid`);
            sshConfigs.push({
                sshHost: host,
                config: {
                    host: host.hostIp,
                    username: host.username,
                    identity: host.identityFile,
                    port: host.port,
                    passphrase: passphraseKeyFile
                }
            });
        }

        return sshConfigs;
    }

    /**
     * Begin an interactive debugging session (user is prompted with questions)
     * to convert invalid SSH hosts to valid SSH hosts.
     * @param invalidSSHHosts A list of invalid SSH hosts
     * @param onCancel Handler function for interruption of the prompt (e.g.
     * CTRL-C, ESC, etc.)
     * @returns A list of valid SSH hosts (potentially empty)
     */
    public async promptInteractiveDebugSession(invalidSSHHosts: InvalidSSHHost[], onCancel: (prompt: PromptObject, answers: any) => void): Promise<ValidSSHHost[]> {
        // Get pretty string of invalid SSH hosts' names
        const prettyInvalidSSHHosts: string = invalidSSHHosts.map(host => host.incompleteValidSSHHost.name).join(', ');
        this.logger.warn(`Hosts missing required parameters: ${prettyInvalidSSHHosts}`);

        const fixedSSHHosts: ValidSSHHost[] = [];
        const confirmDebugSessionResponse = await prompts({
            type: 'toggle',
            name: 'value',
            message: 'Do you want the zli to help you fix the issues?',
            initial: true,
            active: 'yes',
            inactive: 'no',
        }, { onCancel: onCancel });
        const shouldStartDebugSession: boolean = confirmDebugSessionResponse.value;

        if (!shouldStartDebugSession)
            return fixedSSHHosts;

        this.logger.info('\nPress CTRL-C to skip the prompted host or to exit out of quickstart\n');

        for (const invalidSSHHost of invalidSSHHosts) {
            const fixedHost = await this.promptFixInvalidSSHHost(invalidSSHHost);
            if (fixedHost === undefined) {
                const shouldSkip = await this.promptSkipHostOrExit(invalidSSHHost.incompleteValidSSHHost.name, onCancel);

                if (shouldSkip) {
                    this.logger.info(`Skipping host ${invalidSSHHost.incompleteValidSSHHost.name}...`);
                    continue;
                } else {
                    this.logger.info('Prompt cancelled. Exiting out of quickstart...');
                    await cleanExit(1, this.logger);
                }
            } else {
                fixedSSHHosts.push(fixedHost);
            }
        }

        return fixedSSHHosts;
    }

    /**
     * Prompt the user with a series of questions to convert an invalid SSH host
     * to a valid SSH host
     * @param invalidSSHHost An invalid SSH host
     * @returns A valid SSH host if the user answered all prompts and did not
     * cancel. Undefined if the user cancelled any prompt.
     */
    private async promptFixInvalidSSHHost(invalidSSHHost: InvalidSSHHost): Promise<ValidSSHHost | undefined> {
        const parseErrors = invalidSSHHost.parseErrors;
        const sshHostName = invalidSSHHost.incompleteValidSSHHost.name;

        this.logger.info(`Please answer the following ${parseErrors.length} question(s) so that ${sshHostName} can be considered as a valid host to connect with BastionZero`);

        // Iterate through all parse errors for the passed in host and prompt
        // user to fix the problem.
        //
        // If the prompt is cancelled, undefined will be returned. We check for
        // this on each missing parameter, and return undefined as the return
        // value in order to short-circuit any remaining parse errors.
        const validSSHHost = invalidSSHHost.incompleteValidSSHHost;
        for (const parseError of parseErrors) {
            switch (parseError.error) {
            case 'missing_host_name':
                const hostName = await this.handleMissingHostName();
                if (hostName === undefined) {
                    return undefined;
                } else {
                    validSSHHost.hostIp = hostName;
                }
                break;
            case 'missing_port':
                const port = await this.handleMissingPort();
                if (port === undefined) {
                    return undefined;
                } else {
                    validSSHHost.port = port;
                }
                break;
            case 'missing_user':
                const user = await this.handleMissingUser();
                if (user === undefined) {
                    return undefined;
                } else {
                    validSSHHost.username = user;
                }
                break;
            case 'missing_identity_file':
                const identityFilePath = await this.handleMissingIdentityFile();
                if (identityFilePath === undefined) {
                    return undefined;
                } else {
                    validSSHHost.identityFile = this.resolveHome(identityFilePath);
                }
                break;
            default:
                // Note: This error is never thrown at runtime. It is an
                // exhaustive check at compile-time.
                const exhaustiveCheck: never = parseError;
                throw new Error(`Unhandled parse error type: ${exhaustiveCheck}`);
            }
        }

        return validSSHHost;
    }

    /**
     * Prompt the user to provide their passphrase to handle an encrypted
     * identity file (SSH key).
     * @param identityFilePathName File path to key
     * @param identityFileContents Contents of the key
     * @returns A valid passphrase that correctly decrypts the SSH key.
     * Undefined if the user cancels the prompt.
     */
    private async handleEncryptedIdentityFile(identityFilePathName: string, identityFileContents: string): Promise<string | undefined> {
        return new Promise<string | undefined>(async (resolve, _) => {
            const onCancel = () => resolve(undefined);
            const onSubmit = (_: PromptObject, answer: string) => resolve(answer);

            // Custom validation function. Require value to be passed and check
            // if password is correct
            const onValidate = (value: any): string | boolean => {
                if (value) {
                    try {
                        parsePrivateKey(identityFileContents, 'auto', { passphrase: value });

                        // Password is correct!
                        return true;
                    } catch (err) {
                        // Password is either wrong or there was some error when reading the IdentityFile
                        return 'Failed reading file with provided passphrase. Use CTRL-C to skip this host';
                    }
                } else {
                    return 'Value is required. Use CTRL-C to skip this host';
                }
            };

            await prompts({
                type: 'password',
                name: 'value',
                message: `Enter the passphrase for the encrypted SSH key ${identityFilePathName}:`,
                validate: onValidate
            }, { onSubmit: onSubmit, onCancel: onCancel });
        });
    }

    /**
     * Prompt the user to handle a missing HostName parameter for some parsed
     * SSH host.
     * @returns User's answer. Undefined if the user cancels the prompt.
     */
    private async handleMissingHostName(): Promise<string | undefined> {
        return new Promise<string | undefined>(async (resolve, _) => {
            const onCancel = () => resolve(undefined);
            const onSubmit = (_: PromptObject, answer: string) => resolve(answer);
            await prompts({
                type: 'text',
                name: 'value',
                message: 'Enter HostName (IP address or DNS name):',
                validate: value => value ? true : 'Value is required. Use CTRL-C to skip this host'
            }, { onSubmit: onSubmit, onCancel: onCancel });
        });
    }

    /**
     * Prompt the user to handle a missing Port parameter for some parsed SSH
     * host. Prompt defaults to port 22.
     * @returns User's answer. Undefined if the user cancels the prompt.
     */
    private async handleMissingPort(): Promise<number | undefined> {
        return new Promise<number | undefined>(async (resolve, _) => {
            const onCancel = () => resolve(undefined);
            const onSubmit = (_: PromptObject, answer: number) => resolve(answer);
            await prompts({
                type: 'number',
                name: 'value',
                message: 'Enter Port number (default 22):',
                initial: 22,
            }, { onSubmit: onSubmit, onCancel: onCancel });
        });
    }

    /**
     * Prompt the user to handle a missing User parameter for some parsed SSH
     * host.
     * @returns User's answer. Undefined if the user cancels the prompt.
     */
    private async handleMissingUser(): Promise<string | undefined> {
        return new Promise<string | undefined>(async (resolve, _) => {
            const onCancel = () => resolve(undefined);
            const onSubmit = (_: PromptObject, answer: string) => resolve(answer);
            await prompts({
                type: 'text',
                name: 'value',
                message: 'Enter User:',
                validate: value => value ? true : 'Value is required. Use CTRL-C to skip this host'
            }, { onSubmit: onSubmit, onCancel: onCancel });
        });
    }

    /**
     * Prompt the user to handle a missing IdentityFile (SSH key) parameter for
     * some parsed SSH host.
     * @returns User's answer. Undefined if the user cancels the prompt.
     */
    private async handleMissingIdentityFile(): Promise<string | undefined> {
        return new Promise<string | undefined>(async (resolve, _) => {
            const onCancel = () => resolve(undefined);
            const onSubmit = (_: PromptObject, answer: string) => resolve(answer);
            await prompts({
                type: 'text',
                name: 'value',
                message: 'Enter path to IdentityFile:',
                validate: value => value ? true : 'Value is required. Use CTRL-C to skip this host'
            }, { onSubmit: onSubmit, onCancel: onCancel });
        });
    }

    /**
     * Create an environment following the quickstart format (description +
     * cleanupTimeout)
     * @param sshUsername Username that all hosts in this environment should
     * have TargetConnect policies for (with TargetUser=sshUsername) once
     * quickstart finishes successfully
     * @param envName Name of environment
     * @returns Returns the ID of the newly created environment
     */
    private async createQuickstartEnvironment(sshUsername: string, envName: string): Promise<string> {
        const createEnvResp = await this.environmentService.CreateEnvironment({
            name: envName,
            description: `Quickstart autogenerated environment for ${sshUsername} users`,
            // This is the default timeout used in the webapp
            offlineCleanupTimeoutHours: 24 * 90
        });
        return createEnvResp.id;
    }

    /**
    * Create a TargetConnect policy that permits:
    * - Subject: The user running quickstart
    * - Action: To perform the following three verbs with TargetUser (unix
        username) == the parsed SSH username: open a shell connection, create an
        SSH tunnel, and perform FUD.
    * - Context: To any target in the environment as specified by envId
     * @param sshUsername Username to use for the TargetUser parameter
     * @param envId ID of environment in which this policy applies to
     * @param policyName Name of the policy
     * @returns Summary of the newly created policy
     */
    private async createQuickstartTargetConnectPolicy(sshUsername: string, envId: string, policyName: string): Promise<PolicySummary> {
        const environmentContext: { [key: string]: PolicyEnvironment } = { [envId]: { id: envId } };
        const targetUserContext: { [key: string]: PolicyTargetUser } = { [sshUsername]: { userName: sshUsername } };
        const verbContext: { [key: string]: Verb } = {
            [VerbType.Shell]: { type: VerbType.Shell },
            [VerbType.Tunnel]: { type: VerbType.Tunnel },
            [VerbType.FileTransfer]: { type: VerbType.FileTransfer }
        };
        const connectContext: TargetConnectContext = {
            targets: undefined,
            environments: environmentContext,
            targetUsers: targetUserContext,
            verbs: verbContext
        };
        const userAsSubject: Subject = {
            id: this.configService.me().id,
            type: SubjectType.User
        };

        return await this.policyService.AddPolicy({
            name: policyName,
            type: PolicyType.TargetConnect.toString(),
            subjects: [userAsSubject],
            groups: [],
            context: JSON.stringify(connectContext),
            policyMetadata: { description: `Quickstart autogenerated policy for ${sshUsername} users` }
        });
    }

    /**
     * Creates an environment for each unique SSH username in the parsed SSH
     * hosts. If the environment already exists, no environment will be created.
     * @param hostsToAdd A list of valid SSH hosts in which environments should
     * be created for
     * @returns A list of registrable SSH hosts. Each registrable host includes
     * the environment ID in which the host should be registered in during the
     * autodiscovery process. The returned list may be smaller than the input
     * list if the request to create the environment fails.
     */
    public async createEnvForUniqueUsernames(hostsToAdd: ValidSSHHostAndConfig[]): Promise<RegistrableSSHHost[]> {
        const registrableSSHHosts: RegistrableSSHHost[] = [];
        const usernameMap: Map<string, ValidSSHHostAndConfig[]> = new Map();

        // Build map of common SSH usernames among the hosts that are expected
        // to be successfully added to BastionZero
        for (const host of hostsToAdd) {
            // Normalize to lowercase
            const usernameMatch = host.sshHost.username.toLowerCase();
            if (usernameMap.has(usernameMatch)) {
                // Update the list with the matching host
                const matchingTargets = usernameMap.get(usernameMatch);
                matchingTargets.push(host);
            } else {
                // Otherwise create new list starting with one host
                usernameMap.set(usernameMatch, [host]);
            }
        }

        // Create an environment per common SSH username
        for (const [username, hosts] of usernameMap) {
            const quickstartEnvName = `${username}-users_quickstart`;
            let quickstartEnvId: string;
            try {
                const envs = await this.environmentService.ListEnvironments();
                const quickstartEnv = envs.find(env => env.name === quickstartEnvName);
                if (quickstartEnv === undefined) {
                    // Quickstart env for this ssh username does not exist

                    // Create new environment
                    quickstartEnvId = await this.createQuickstartEnvironment(username, quickstartEnvName);
                } else {
                    // Environment already exists
                    quickstartEnvId = quickstartEnv.id;
                }

                // Convert hosts to registrable hosts with accompanying
                // environment id to use during registration
                hosts.forEach(host => registrableSSHHosts.push({ host: host, envId: quickstartEnvId }));
            } catch (err) {
                this.logger.error(`Failed creating env for SSH username ${username}: ${err}`);
                continue;
            }
        }

        return registrableSSHHosts;
    }

    /**
     * Creates a policy for each unique SSH username in the parsed SSH hosts. If
     * the policy already exists, no new policy will be created.
     * @param quickstartTargets A list of SSM targets that were successfully
     * registered
     * @returns A list of SSM targets which are most likely connectable. The
     * list will be smaller than the input list if the request to create the
     * policy fails.
     *
     * We say that the hosts are most likely connectable because there is no
     * absolute guarantee that the created policy (or the retrieved policy in
     * the case that the policy already exists) will permit the user running
     * quickstart to connect to their targets. Some bad scenarios include:
     * - There exists a policy with the expected name, but it does not actually
     *   permit the user running quickstart to connect.
     * - The created policy (in the case that no expected policy exists) is
     *   changed or removed before quickstart actually makes the connection.
     */
    public async createPolicyForUniqueUsernames(quickstartTargets: QuickstartSSMTarget[]): Promise<QuickstartSSMTarget[]> {
        const connectableTargets: QuickstartSSMTarget[] = [];
        const usernameMap: Map<string, QuickstartSSMTarget[]> = new Map();

        // Build map of common SSH usernames among the targets that were
        // successfully added to BastionZero
        for (const target of quickstartTargets) {
            // Normalize to lowercase
            const usernameMatch = target.sshHost.username.toLowerCase();
            if (usernameMap.has(usernameMatch)) {
                // Update the list with the matching target
                const matchingTargets = usernameMap.get(usernameMatch);
                matchingTargets.push(target);
            } else {
                // Otherwise create new list starting with one target
                usernameMap.set(usernameMatch, [target]);
            }
        }

        // Create a policy per common SSH username
        for (const [username, targets] of usernameMap) {
            const quickstartPolicyName = `${username}-users-policy_quickstart`;
            try {
                // Ensure that quickstart policy exists for this SSH username.
                const policies = await this.policyService.ListAllPolicies();
                if (policies.find(policy => policy.name === quickstartPolicyName) === undefined) {
                    // Quickstart policy for this ssh username does not exist

                    // All targets with the same SSH username were registered in
                    // the same environment
                    const envId = targets[0].ssmTarget.environmentId;

                    // Create new policy
                    await this.createQuickstartTargetConnectPolicy(username, envId, quickstartPolicyName);
                }

                // Either the policy already exists, or we've just successfully
                // created one. Add all targets to final list of
                // connectableTargets.
                //
                // NOTE: It's entirely possible that even though the policy
                // exists with the correct name above, the policy has been
                // changed in such a way that shell connect becomes impossible
                // for these targets (e.g. current user removed from list of
                // subjects, connect verb removed, etc.). We've chosen not to
                // cover this edge case.
                targets.forEach(target => connectableTargets.push(target));
            } catch (err) {
                this.logger.error(`Failed creating policy for SSH username ${username}: ${err}`);
                continue;
            }
        }

        return connectableTargets;
    }

    /**
     * Parse SSH hosts from a valid ssh_config(5)
     * (https://linux.die.net/man/5/ssh_config)
     * @param sshConfig Contents of the ssh config file
     * @returns A 2-tuple.
     *
     * The first element contains a mapping of all valid SSH hosts. The key is
     * the SSH host's name. The value is an interface containing information
     * about the host. A valid SSH host is defined as one that has enough
     * information about it in the parsed config file, so that it can be used
     * with the ssh2-promise library. There is no guarantee that a valid ssh
     * host is successfully connectable (e.g. network issue, encrypted key file,
     * invalid IP/host, file not found at path, etc.).
     *
     * The second element contains a list of all invalid SSH hosts. Each invalid
     * SSH host contains an associated list of parse errors and an incomplete
     * ValidSSHHost (e.g. some required parameters were included).
     */
    public parseSSHHosts(sshConfig: string): [hosts: Map<string, ValidSSHHost>, invalidSSHHosts: InvalidSSHHost[]] {
        // Parse sshConfig content to usable HostBlock types
        const SSHConfig = require('ssh-config');
        const config: [] = SSHConfig.parse(sshConfig);
        const hostBlocks: SSHConfigHostBlock[] = config.filter((elem: any) => elem.param && elem.param.toLowerCase() === 'host');

        const seen: Map<string, boolean> = new Map();
        const validHosts: Map<string, ValidSSHHost> = new Map();
        const invalidSSHHosts: InvalidSSHHost[] = [];

        for (const hostBlock of hostBlocks) {
            const name = hostBlock.value;
            // Skip global directive
            if (name === '*') {
                continue;
            }

            // Skip host if already found. Print warning to user. This behavior
            // is on par with how ssh works with duplicate hosts (the first host
            // is used and the second is skipped).
            if (seen.has(name)) {
                this.logger.warn(`Notice: Already seen SSH host with Host == ${name}. Keeping the first one seen.`);
                continue;
            }
            seen.set(name, true);

            // Rolling build of valid SSH host
            const validSSHHost = {} as ValidSSHHost;
            validSSHHost.name = name;

            // Array holds all config parse errors found while parsing
            const parseErrors: SSHConfigParseError[] = [];
            const config = hostBlock.config;

            // Parse required SSH config parameters
            const hostIp = this.getSSHHostConfigValue('HostName', config);
            if (hostIp === undefined) {
                parseErrors.push({ error: 'missing_host_name' });
            } else {
                validSSHHost.hostIp = hostIp;
            }
            const port = this.getSSHHostConfigValue('Port', config);
            if (port === undefined) {
                parseErrors.push({ error: 'missing_port' });
            } else {
                validSSHHost.port = parseInt(port);
            }
            const user = this.getSSHHostConfigValue('User', config);
            if (user === undefined) {
                parseErrors.push({ error: 'missing_user' });
            } else {
                validSSHHost.username = user;
            }
            const identityFilePath = this.getSSHHostConfigValue('IdentityFile', config);
            if (identityFilePath === undefined) {
                parseErrors.push({ error: 'missing_identity_file' });
            } else {
                validSSHHost.identityFile = this.resolveHome(identityFilePath);
            }

            if (parseErrors.length > 0) {
                invalidSSHHosts.push({
                    incompleteValidSSHHost: validSSHHost,
                    parseErrors: parseErrors
                });
                this.logger.debug(`Failed to parse host: ${name}`);
                continue;
            }

            validHosts.set(name, validSSHHost);
        }

        return [validHosts, invalidSSHHosts];
    }

    private getSSHHostConfigValue(matchingParameter: string, hostConfig: SSHHostConfig[]): string | undefined {
        const value = hostConfig.find(elem => elem.param && elem.param.toLowerCase() === matchingParameter.toLowerCase());
        if (value === undefined) {
            return undefined;
        } else {
            return value.value;
        }
    }

    private resolveHome(filepath: string) {
        if (filepath[0] === '~') {
            return path.join(os.homedir(), filepath.slice(1));
        }
        return filepath;
    }
}