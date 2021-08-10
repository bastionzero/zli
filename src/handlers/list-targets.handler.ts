import {
    findSubstring,
    parseTargetType,
    getTableOfTargets,
    parseTargetStatus
} from '../utils';
import { Logger } from '../logger.service/logger';
import { EnvironmentDetails } from '../http.service/http.service.types';
import { cleanExit } from './clean-exit.handler';
import { SsmTargetStatus, TargetSummary, TargetType } from '../types';
import _ from 'lodash';


export async function listTargetsHandler(
    logger: Logger,
    argv: any,
    dynamicConfigs: Promise<TargetSummary[]>,
    ssmTargets: Promise<TargetSummary[]>,
    envsPassed: Promise<EnvironmentDetails[]>) {
    // await and concatenate
    let allTargets = [...await ssmTargets, ...await dynamicConfigs];
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

    if(!! argv.status) {
        const statusArray: string[] = argv.status;

        if(statusArray.length < 1) {
            logger.warn('Status filter flag passed with no arguments, please indicate at least one status');
            await cleanExit(1, logger);
        }

        let targetStatusFilter: SsmTargetStatus[] = _.map(statusArray, (s: string) => parseTargetStatus(s)).filter(s => s); // filters out undefined
        targetStatusFilter = _.uniq(targetStatusFilter);

        if(targetStatusFilter.length < 1) {
            logger.warn('Status filter flag passed with no valid arguments, please indicate at least one valid status');
            await cleanExit(1, logger);
        }

        allTargets = allTargets.filter(t => t.type != TargetType.SSM || _.includes(targetStatusFilter, t.status));
    }

    if(!! argv.json) {
        // json output
        console.log(JSON.stringify(allTargets));
    } else {
        // regular table output
        // We OR the detail and status flags since we want to show the details in both cases
        const tableString = getTableOfTargets(allTargets, envs, !! argv.detail || !! argv.status, !! argv.showId);
        console.log(tableString);
    }

    await cleanExit(0, logger);
}