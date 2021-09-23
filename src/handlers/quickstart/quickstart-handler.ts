import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { cleanExit } from '../clean-exit.handler';
import { QuickstartSsmService } from '../../services/quickstart/quickstart-ssm.service';
import { ValidSSHHost } from '../../services/quickstart/quickstart-ssm.service.types';
import { MixpanelService } from '../../services/mixpanel/mixpanel.service';
import { ParsedTargetString, TargetType } from '../../services/common.types';
import { EnvironmentService } from '../../services/environment/environment.service';
import { PolicyService } from '../../services/policy/policy.service';
import { connectHandler } from '../connect/connect.handler';
import { readFile } from '../../utils';
import { SsmTargetSummary } from '../../services/ssm-target/ssm-target.types';
import { defaultSshConfigFilePath, quickstartArgs } from './quickstart.command-builder';

import prompts from 'prompts';
import yargs from 'yargs';
import fs from 'fs';

async function validateQuickstartArgs(argv: yargs.Arguments<quickstartArgs>) {
    // OS check
    if (process.platform === 'win32') {
        throw new Error('Quickstart is not supported on Windows machines');
    }

    // Check sshConfigFile parameter
    if (argv.sshConfigFile === undefined) {
        // User did not pass in sshConfigFile parameter. Use default parameter
        argv.sshConfigFile = defaultSshConfigFilePath;
        if (!fs.existsSync(argv.sshConfigFile)) {
            throw new Error(`Cannot read/access file at default path: ${argv.sshConfigFile}\nUse \`zli quickstart --sshConfigFile <filePath>\` to read a different file`);
        }
    } else {
        // User passed in sshConfigFile
        if (!fs.existsSync(argv.sshConfigFile)) {
            throw new Error(`Cannot read/access file at path: ${argv.sshConfigFile}`);
        }
    }
}

export async function quickstartHandler(
    argv: yargs.Arguments<quickstartArgs>,
    logger: Logger,
    configService: ConfigService,
    mixpanelService: MixpanelService,
) {
    await validateQuickstartArgs(argv);

    const policyService = new PolicyService(configService, logger);
    const envService = new EnvironmentService(configService, logger);
    const quickstartService = new QuickstartSsmService(logger, configService, policyService, envService);

    // Parse SSH config file
    logger.info(`\nParsing SSH config file: ${argv.sshConfigFile}`);
    const sshConfigFileAsStr = await readFile(argv.sshConfigFile);
    const [validSSHHosts, invalidSSHHosts] = quickstartService.parseSSHHosts(sshConfigFileAsStr);

    // Callback on cancel prompt
    const onCancelPrompt = async () => {
        logger.info('Prompt cancelled. Exiting out of quickstart...');
        await cleanExit(1, logger);
    };

    logger.info(`\nFound ${validSSHHosts.size} valid SSH hosts!`);

    // Handle parse errors
    if (invalidSSHHosts.length > 0) {
        logger.warn(`${invalidSSHHosts.length} host(s) in the SSH config file are missing required parameters used to connect them with BastionZero.`);
        const fixedSSHHosts = await quickstartService.promptInteractiveDebugSession(invalidSSHHosts, onCancelPrompt);
        // Add them to the valid mapping
        fixedSSHHosts.forEach(validHost => validSSHHosts.set(validHost.name, validHost));
        if (fixedSSHHosts.length > 0) {
            logger.info(`Added ${fixedSSHHosts.length} more valid host(s) for a total of ${validSSHHosts.size} valid SSH hosts!`);
        }
    }

    // Fail early if there are no valid hosts to choose from
    if (validSSHHosts.size == 0) {
        logger.error('Exiting because there are no valid hosts to connect to');
        await cleanExit(1, logger);
    }

    logger.info('\nPress CTRL-C to exit at any time.');

    // Prompt user with selection of hosts
    const hostsResponse = await prompts({
        type: 'multiselect',
        name: 'value',
        message: 'Which SSH hosts do you want to connect with BastionZero?',
        choices: Array.from(validSSHHosts.keys()).map(hostName => ({ title: hostName, value: hostName } as prompts.Choice)),
        instructions: 'Use space to select and up/down to navigate. Return to submit.'
    }, { onCancel: onCancelPrompt });
    const selectedHostsNames: string[] = hostsResponse.value;

    if (selectedHostsNames.length == 0) {
        logger.info('No hosts selected. Exiting out of quickstart...');
        await cleanExit(1, logger);
    }

    // Ask user if they want to connect to one of their target(s) after it is
    // registered
    const connectAfterResponse = await prompts({
        type: 'toggle',
        name: 'value',
        message: 'Do you want to immediately connect to your target once it is registered with BastionZero?',
        initial: true,
        active: 'yes',
        inactive: 'no',
    }, { onCancel: onCancelPrompt });
    const shouldConnectAfter: boolean = connectAfterResponse.value;
    let targetToConnectToAtEnd: ValidSSHHost = undefined;
    if (shouldConnectAfter && selectedHostsNames.length > 1) {
        // If the user selected more than one host, then ask which host they
        // want to connect to
        const choices = selectedHostsNames.map(hostName => ({ title: hostName, value: hostName } as prompts.Choice));
        const targetToConnectAfterResponse = await prompts({
            type: 'select',
            name: 'value',
            message: 'Which target?',
            choices: choices,
            initial: 1,
            instructions: 'Use up/down to navigate. Use tab to cycle the list. Return to submit.'
        }, { onCancel: onCancelPrompt });
        targetToConnectToAtEnd = validSSHHosts.get(targetToConnectAfterResponse.value);
    }
    else if (shouldConnectAfter && selectedHostsNames.length == 1) {
        // Otherwise, we know which host it is
        targetToConnectToAtEnd = validSSHHosts.get(selectedHostsNames[0]);
    }

    // Convert list of selected ValidSSHHosts to SSHConfigs to use with the
    // ssh2-promise library. This conversion is interactive. It will prompt the
    // user to provide a passphrase if any of the selected hosts' IdentityFiles
    // are encrypted.
    const validSSHConfigs = await quickstartService.promptConvertValidSSHHostsToSSHConfigs(
        selectedHostsNames.map(hostName => validSSHHosts.get(hostName)),
        onCancelPrompt);

    // Fail early if the validation check above removed all valid hosts
    if (validSSHConfigs.length == 0) {
        logger.info('All selected hosts were removed from list of SSH hosts to add to BastionZero. Exiting out of quickstart...');
        await cleanExit(1, logger);
    }

    // Ask the user if they're ready to begin
    const prettyHostsToAttemptAutodisocvery: string = validSSHConfigs.map(config => `\t- ${config.sshHost.name}`).join('\n');
    const readyResponse = await prompts({
        type: 'toggle',
        name: 'value',
        message: `Please confirm that you want to add:\n\n${prettyHostsToAttemptAutodisocvery}\n\nto BastionZero:`,
        initial: true,
        active: 'yes',
        inactive: 'no',
    }, { onCancel: onCancelPrompt });
    const isReady: boolean = readyResponse.value;

    if (!isReady) {
        logger.info('Exiting out of quickstart...');
        await cleanExit(1, logger);
    }

    // Create environment for each unique username parsed from the SSH config
    const registrableHosts = await quickstartService.createEnvForUniqueUsernames(validSSHConfigs);

    // Run autodiscovery script on all hosts concurrently
    const autodiscoveryResultsPromise = Promise.allSettled(registrableHosts.map(host => quickstartService.addSSHHostToBastionZero(host)));

    // Await for **all** hosts to either come "Online" or error
    const autodiscoveryResults = await autodiscoveryResultsPromise;
    const ssmTargetsSuccessfullyAdded = autodiscoveryResults.reduce<SsmTargetSummary[]>((acc, result) => result.status === 'fulfilled' ? [...acc, result.value] : acc, []);

    // Exit early if all hosts failed
    if (ssmTargetsSuccessfullyAdded.length == 0) {
        await cleanExit(1, logger);
    }

    // Create policy for each unique username parsed from the SSH config
    const connectableTargets = await quickstartService.createPolicyForUniqueUsernames(
        ssmTargetsSuccessfullyAdded.map(target => ({ ssmTarget: target, sshHost: validSSHHosts.get(target.name) }))
    );
    const isAtLeastOneConnectableTarget = connectableTargets.length > 0;

    // Gather extra information if user said to connect to specific
    // target after registration completes.
    let targetToConnectToAtEndAsParsedTargetString: ParsedTargetString = undefined;
    if (shouldConnectAfter) {
        // targetToConnectToAtEnd is guaranteed to be defined if shouldConnectAfter == true
        const ssmTargetToConnectToAtEnd = connectableTargets.find(target => target.ssmTarget.name === targetToConnectToAtEnd.name);
        if (ssmTargetToConnectToAtEnd) {
            const envs = await envService.ListEnvironments();
            const environment = envs.find(envDetails => envDetails.id == ssmTargetToConnectToAtEnd.ssmTarget.environmentId);
            targetToConnectToAtEndAsParsedTargetString = {
                id: ssmTargetToConnectToAtEnd.ssmTarget.id,
                user: ssmTargetToConnectToAtEnd.sshHost.username,
                type: TargetType.SSM,
                envName: environment.name
            } as ParsedTargetString;
        }
    }

    let exitCode = isAtLeastOneConnectableTarget ? 0 : 1;
    if (targetToConnectToAtEndAsParsedTargetString) {
        logger.info(`Connecting to ${targetToConnectToAtEnd.name} by using \`zli connect ${targetToConnectToAtEndAsParsedTargetString.user}@${targetToConnectToAtEnd.name}\``);
        exitCode = await connectHandler(configService, logger, mixpanelService, targetToConnectToAtEndAsParsedTargetString);
    }

    if (isAtLeastOneConnectableTarget) {
        logger.info('Use `zli connect` to connect to your newly registered targets.');
        for (const target of connectableTargets) {
            logger.info(`\tzli connect ${target.sshHost.username}@${target.ssmTarget.name}`);
        }
    }

    await cleanExit(exitCode, logger);
}