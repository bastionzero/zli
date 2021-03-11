import {
    TargetSummary,
    findSubstring,
    parseTargetType,
    getTableOfTargets
} from '../utils';
import { Logger } from '../logger.service/logger';
import { EnvironmentDetails } from '../http.service/http.service.types';
import { cleanExit } from './clean-exit.handler';


export async function listTargetsHandler(
    logger: Logger,
    argv: any,
    dynamicConfigs: Promise<TargetSummary[]>,
    ssmTargets: Promise<TargetSummary[]>,
    sshTargets: Promise<TargetSummary[]>,
    envsPassed: Promise<EnvironmentDetails[]>) {
    // await and concatenate
    let allTargets = [...await ssmTargets, ...await sshTargets, ...await dynamicConfigs];
    let envs = await envsPassed;

    // find all envIds with substring search
    // filter targets down by endIds
    // ref for '!!': https://stackoverflow.com/a/29312197/14782428
    if(!! argv.env)
    {
        const envIdFilter = envs.filter(e => findSubstring(argv.env, e.name)).map(e => e.id);

        allTargets = allTargets.filter(t => envIdFilter.includes(t.environmentId));
    }

    // filter targets by name/alias
    if(!! argv.name)
    {
        allTargets = allTargets.filter(t => findSubstring(argv.name, t.name));
    }

    // filter targets by TargetType
    if(!! argv.targetType)
    {
        let targetType = parseTargetType(argv.targetType);
        allTargets = allTargets.filter(t => t.type === targetType);
    }

    let tableString = getTableOfTargets(allTargets, envs);
    console.log(tableString);
    await cleanExit(0, logger);
}