import { ConfigService } from '../../services/config/config.service';
import { Logger } from '../../services/logger/logger.service';
import { cleanExit } from '../clean-exit.handler';
import { QuickstartSsmService } from '../../services/quickstart/quickstart-ssm.service';
import { InvalidSSHHost, ValidSSHConfig, ValidSSHHost } from '../../services/quickstart/quickstart-ssm.service.types';
import { MixpanelService } from '../../services/mixpanel/mixpanel.service';
import { ParsedTargetString, TargetType } from '../../services/common.types';
import { EnvironmentService } from '../../services/environment/environment.service';
import { connectHandler } from '../connect/connect.handler';
import { readFile } from '../../utils';
import { SsmTargetSummary } from '../../services/ssm-target/ssm-target.types';
import { quickstartArgs } from './quickstart.command-builder';

import prompts, { PromptObject } from 'prompts';
import yargs from 'yargs';

async function interactiveDebugSession(
    invalidSSHHosts: InvalidSSHHost[],
    quickstartService: QuickstartSsmService,
    logger: Logger,
    onCancel: (prompt: PromptObject, answers: any) => void): Promise<ValidSSHHost[]> {

    // Get pretty string of invalid SSH hosts' names
    const prettyInvalidSSHHosts: string = invalidSSHHosts.map(host => host.incompleteValidSSHHost.name).join(', ');
    logger.warn(`Hosts missing required parameters: ${prettyInvalidSSHHosts}`);

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

    logger.info('\nPress CTRL-C to skip the prompted host or to exit out of quickstart\n');

    for (const invalidSSHHost of invalidSSHHosts) {
        const fixedHost = await quickstartService.promptFixInvalidSSHHost(invalidSSHHost);
        if (fixedHost === undefined) {
            const shouldSkip = await quickstartService.promptSkipHostOrExit(invalidSSHHost.incompleteValidSSHHost.name, onCancel);

            if (shouldSkip) {
                logger.info(`Skipping host ${invalidSSHHost.incompleteValidSSHHost.name}...`);
                continue;
            } else {
                logger.info('Prompt cancelled. Exiting out of quickstart...');
                await cleanExit(1, logger);
            }
        } else {
            fixedSSHHosts.push(fixedHost);
        }
    }

    return fixedSSHHosts;
}

export async function quickstartHandler(
    argv: yargs.Arguments<quickstartArgs>,
    logger: Logger,
    configService: ConfigService,
    mixpanelService: MixpanelService,
) {
    const quickstartService = new QuickstartSsmService(logger, configService);

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
        const fixedSSHHosts = await interactiveDebugSession(invalidSSHHosts, quickstartService, logger, onCancelPrompt);
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

    // Run autodiscovery script on all hosts concurrently
    const autodiscoveryResultsPromise = Promise.allSettled(validSSHConfigs.map(config => addSSHHostToBastionZero(config, quickstartService, logger)));

    // Await for **all** hosts to either come "Online" or error
    const autodiscoveryResults = await autodiscoveryResultsPromise;
    const ssmTargetsSuccessfullyAdded = autodiscoveryResults.reduce<SsmTargetSummary[]>((acc, result) => {
        if (result.status === 'fulfilled') {
            acc.push(result.value);
            return acc;
        } else {
            return acc;
        }
    }, []);
    const didRegisterAtLeastOne = ssmTargetsSuccessfullyAdded.length > 0;

    // Gather extra information if user said to connect to specific
    // target after registration completes.
    let targetToConnectToAtEndAsParsedTargetString: ParsedTargetString = undefined;
    if (shouldConnectAfter) {
        // targetToConnectToAtEnd is guaranteed to be defined if shouldConnectAfter == true
        const ssmTargetToConnectToAtEnd = ssmTargetsSuccessfullyAdded.find(target => target.name === targetToConnectToAtEnd.name);
        if (ssmTargetToConnectToAtEnd) {
            const envService = new EnvironmentService(configService, logger);
            const envs = await envService.ListEnvironments();
            const environment = envs.find(envDetails => envDetails.id == ssmTargetToConnectToAtEnd.environmentId);
            targetToConnectToAtEndAsParsedTargetString = {
                id: ssmTargetToConnectToAtEnd.id,
                user: 'ssm-user',
                type: TargetType.SSM,
                envName: environment.name
            } as ParsedTargetString;
        }
    }

    let exitCode = didRegisterAtLeastOne ? 0 : 1;
    if (targetToConnectToAtEndAsParsedTargetString) {
        logger.info(`Connecting to ${targetToConnectToAtEnd.name} by using \`zli connect ${targetToConnectToAtEndAsParsedTargetString.user}@${targetToConnectToAtEnd.name}\``);
        exitCode = await connectHandler(configService, logger, mixpanelService, targetToConnectToAtEndAsParsedTargetString);
    }

    if (didRegisterAtLeastOne) {
        logger.info('Use `zli connect` to connect to your registered targets.');
        for (const ssmTarget of ssmTargetsSuccessfullyAdded) {
            logger.info(`\tzli connect ssm-user@${ssmTarget.name}`);
        }
    }

    await cleanExit(exitCode, logger);
}

async function addSSHHostToBastionZero(
    validSSHConfig: ValidSSHConfig,
    quickstartService: QuickstartSsmService,
    logger: Logger): Promise<SsmTargetSummary> {

    return new Promise<SsmTargetSummary>(async (resolve, reject) => {
        try {
            logger.info(`Attempting to add SSH host ${validSSHConfig.sshHostName} to BastionZero...`);

            logger.info(`Running autodiscovery script on SSH host ${validSSHConfig.sshHostName} (could take several minutes)...`);
            const ssmTargetId = await quickstartService.runAutodiscoveryOnSSHHost(validSSHConfig.config, validSSHConfig.sshHostName);
            logger.info(`Bastion assigned SSH host ${validSSHConfig.sshHostName} with the following unique target id: ${ssmTargetId}`);

            // Poll for "Online" status
            logger.info(`Waiting for target ${validSSHConfig.sshHostName} to become online (could take several minutes)...`);
            const ssmTarget = await quickstartService.pollSsmTargetOnline(ssmTargetId);
            logger.info(`SSH host ${validSSHConfig.sshHostName} successfully added to BastionZero!`);

            resolve(ssmTarget);
        } catch (error) {
            logger.error(`Failed to add SSH host: ${validSSHConfig.sshHostName} to BastionZero. ${error}`);
            reject(error);
        }
    });
}