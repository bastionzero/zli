import {
    findSubstring,
    parseTargetType,
    getTableOfTargets
} from '../utils';
import { Logger } from '../logger.service/logger';
import { EnvironmentDetails } from '../http.service/http.service.types';
import { cleanExit } from './clean-exit.handler';
import { TargetSummary } from '../types';


export async function listTargetsHandler(
    logger: Logger,
    argv: any,
    dynamicConfigs: Promise<TargetSummary[]>,
    ssmTargets: Promise<TargetSummary[]>,
    sshTargets: Promise<TargetSummary[]>,
    envsPassed: Promise<EnvironmentDetails[]>) {
    // await and concatenate
    let allTargets = [...await ssmTargets, ...await sshTargets, ...await dynamicConfigs];
    const envs = await envsPassed;

    // find all envIds with substring search
    // filter targets down by endIds
    // ref for '!!': https://stackoverflow.com/a/29312197/14782428
    if(!! argv.env) {
        const envIdFilter = envs.filter(e => findSubstring(argv.env, e.name)).map(e => e.id);
        allTargets = allTargets.filter(t => envIdFilter.includes(t.environmentId));
    }

    // filter targets by name/alias
    if(!! argv.name) {
        allTargets = allTargets.filter(t => findSubstring(argv.name, t.name));
    }

    // filter targets by TargetType
    if(!! argv.targetType) {
        const targetType = parseTargetType(argv.targetType);
        allTargets = allTargets.filter(t => t.type === targetType);
    }

    if(!! argv.json) {
        // json output
        console.log(JSON.stringify(allTargets));
    } else {
        // regular table output
        const tableString = getTableOfTargets(allTargets, envs, !! argv.showId);
        console.log(tableString);
    }

    await cleanExit(0, logger);
}