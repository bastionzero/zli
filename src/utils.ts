import { ConnectionDetails, ParsedTargetString, SsmTargetStatus, TargetSummary, TargetType } from './types';
import { max, filter, concat } from 'lodash';
import { EnvironmentDetails } from './http.service/http.service.types';
import Table from 'cli-table3';
import { Logger } from './logger.service/logger';
import { cleanExit } from './handlers/clean-exit.handler';

// case insensitive substring search, 'find targetString in searchString'
export function findSubstring(targetString: string, searchString: string) : boolean
{
    return searchString.toLowerCase().indexOf(targetString.toLowerCase()) !== -1;
}

export const targetStringExample : string = '[targetUser@]<targetId-or-targetName>';

export function parseTargetType(targetType: string) : TargetType
{
    const targetTypePattern = /^(ssm|dynamic)$/i; // case insensitive check for targetType

    if(! targetTypePattern.test(targetType))
        return undefined;

    return <TargetType> targetType.toUpperCase();
}

export function parseTargetStatus(targetStatus: string) : SsmTargetStatus {
    switch (targetStatus.toLowerCase()) {
    case SsmTargetStatus.NotActivated.toLowerCase():
        return SsmTargetStatus.NotActivated;
    case SsmTargetStatus.Offline.toLowerCase():
        return SsmTargetStatus.Offline;
    case SsmTargetStatus.Online.toLowerCase():
        return SsmTargetStatus.Online;
    case SsmTargetStatus.Terminated.toLowerCase():
        return SsmTargetStatus.Terminated;
    case SsmTargetStatus.Error.toLowerCase():
        return SsmTargetStatus.Error;
    default:
        return undefined;
    }
}

export function parseClusterStatus(clusterStatus: string) : KubeClusterStatus {
    switch (clusterStatus.toLowerCase()) {
    case KubeClusterStatus.NotActivated.toLowerCase():
        return KubeClusterStatus.NotActivated;
    case KubeClusterStatus.Offline.toLowerCase():
        return KubeClusterStatus.Offline;
    case KubeClusterStatus.Online.toLowerCase():
        return KubeClusterStatus.Online;
    case KubeClusterStatus.Terminated.toLowerCase():
        return KubeClusterStatus.Terminated;
    case KubeClusterStatus.Error.toLowerCase():
        return KubeClusterStatus.Error;
    default:
        return undefined;
    }
}

export function parseTargetString(targetString: string) : ParsedTargetString
{
    // case sensitive check for [targetUser@]<targetId | targetName>[:targetPath]
    const pattern = /^([a-z_]([a-z0-9_-]{0,31}|[a-z0-9_-]{0,30}\$)@)?(([0-9A-Fa-f]{8}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{12})|([a-zA-Z0-9_.-]{1,255}))(:{1}|$)/;

    if(! pattern.test(targetString))
        return undefined;

    const result : ParsedTargetString = {
        type: undefined,
        user: undefined,
        id: undefined,
        name: undefined,
        path: undefined,
        envId: undefined,
        envName: undefined
    };

    let atSignSplit = targetString.split('@', 2);

    // if targetUser@ is present, extract username
    if(atSignSplit.length == 2)
    {
        result.user = atSignSplit[0];
        atSignSplit = atSignSplit.slice(1);
    }

    // extract targetId and maybe targetPath
    const colonSplit = atSignSplit[0].split(':', 2);
    const targetSomething = colonSplit[0];

    // test if targetSomething is GUID
    if(isGuid(targetSomething))
        result.id = targetSomething;
    else
        result.name = targetSomething;

    if(colonSplit[1] !== '')
        result.path = colonSplit[1];

    return result;
}

// Checks whether the passed argument is a valid Guid
export function isGuid(id: string): boolean{
    const guidPattern = /^[0-9A-Fa-f]{8}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{12}$/;
    return guidPattern.test(id);
}

export function getTableOfTargets(targets: TargetSummary[], envs: EnvironmentDetails[], showDetail: boolean = false, showGuid: boolean = false) : string
{
    const targetNameLength = max(targets.map(t => t.name.length).concat(16)); // if max is 0 then use 16 as width
    const envNameLength = max(envs.map(e => e.name.length).concat(16));       // same same

    const header: string[] = ['Type', 'Name', 'Environment'];
    const columnWidths = [10, targetNameLength + 2, envNameLength + 2];

    if(showGuid)
    {
        header.push('Id');
        columnWidths.push(38);
    }

    if(showDetail)
    {
        header.push('Agent Version', 'Status');
        columnWidths.push(15, 10);
    }

    // ref: https://github.com/cli-table/cli-table3
    const table = new Table({ head: header, colWidths: columnWidths });

    targets.forEach(target => {
        const row = [target.type, target.name, envs.filter(e => e.id == target.environmentId).pop().name];

        if(showGuid) {
            row.push(target.id);
        }

        if(showDetail) {
            row.push(target.agentVersion);
            row.push(target.status || 'N/A'); // status is undefined for non-SSM targets
        }

        table.push(row);
    }
    );

    return table.toString();
}

export function getTableOfClusters(clusters: ClusterSummary[], showDetail: boolean = false, showGuid: boolean = false) : string {
    const clusterNameLength = max(clusters.map(t => t.name.length).concat(16)); // if max is 0 then use 16 as width
    const statusNameLength = max(clusters.map(e => e.status.length).concat(16));
    const header: string[] = ['Cluster Name', 'Status'];
    const columnWidths = [clusterNameLength + 2, statusNameLength + 2];

    if(showGuid || showDetail)
    {
        // For now showGuid and showDetail do the same thing
        header.push('Valid Roles');
        columnWidths.push(38);
        header.push('Id');
        columnWidths.push(38);
        header.push('Agent Version');
        columnWidths.push(38);
        header.push('Last Agent Update');
        columnWidths.push(38);
    }

    // ref: https://github.com/cli-table/cli-table3
    const table = new Table({ head: header, colWidths: columnWidths });

    clusters.forEach(cluster => {
        const row = [cluster.name, cluster.status];
        if (showGuid || showDetail) {
            row.push(cluster.validUsers.toString());
            row.push(cluster.id);
            row.push(cluster.agentVersion);
            row.push(cluster.lastAgentUpdate.toString());
        }
        table.push(row);
    }
    );

    return table.toString();
}

export function getTableOfConnections(connections: ConnectionDetails[], allTargets: TargetSummary[]) : string
{
    const targetNameLength = max(allTargets.map(t => t.name.length).concat(16));
    const connIdLength = max(connections.map(c => c.id.length).concat(36));
    const targetUserLength = max(connections.map(c => c.userName.length).concat(16));
    const header: string[] = ['Connection ID', 'Target User', 'Target', 'Time Created'];
    const columnWidths = [connIdLength + 2, targetUserLength + 2, targetNameLength + 2, 20];

    const table = new Table({ head: header, colWidths: columnWidths });
    const dateOptions = {year: '2-digit', month: 'numeric', day: 'numeric', hour:'numeric', minute:'numeric', hour12: true};
    connections.forEach(connection => {
        const row = [connection.id, connection.userName, allTargets.filter(t => t.id == connection.targetId).pop().name, new Date(connection.timeCreated).toLocaleString('en-US', dateOptions as any)];
        table.push(row);
    });

    return table.toString();

}

// Figure out target id based on target name and target type.
// Also preforms error checking on target type and target string passed in
export async function disambiguateTarget(
    targetTypeString: string,
    targetString: string,
    logger: Logger,
    dynamicConfigs: Promise<TargetSummary[]>,
    ssmTargets: Promise<TargetSummary[]>,
    envs: Promise<EnvironmentDetails[]>): Promise<ParsedTargetString> {

    const parsedTarget = parseTargetString(targetString);

    if(! parsedTarget) {
        return undefined;
    }

    let zippedTargets = concat(await ssmTargets, await dynamicConfigs);

    // Filter out Error and Terminated SSM targets
    zippedTargets = filter(zippedTargets, t => t.type !== TargetType.SSM || (t.status !== SsmTargetStatus.Error && t.status !== SsmTargetStatus.Terminated));

    if(!! targetTypeString) {
        const targetType = parseTargetType(targetTypeString);
        zippedTargets = filter(zippedTargets,t => t.type == targetType);
    }

    let matchedTargets: TargetSummary[];

    if(!! parsedTarget.id) {
        matchedTargets = filter(zippedTargets,t => t.id == parsedTarget.id);
    } else if(!! parsedTarget.name) {
        matchedTargets = filter(zippedTargets,t => t.name == parsedTarget.name);
    }

    if(matchedTargets.length == 0) {
        return undefined;
    } else if(matchedTargets.length == 1) {
        parsedTarget.id = matchedTargets[0].id;
        parsedTarget.name = matchedTargets[0].name;
        parsedTarget.type = matchedTargets[0].type;
        parsedTarget.envId = matchedTargets[0].environmentId;
        parsedTarget.envName = filter(await envs, e => e.id == parsedTarget.envId)[0].name;
    } else {
        logger.warn('More than one target found with the same targetName');
        logger.info(`Please specify the targetId instead of the targetName (zli lt -n ${parsedTarget.name} -d)`);
        await cleanExit(1, logger);
    }

    return parsedTarget;
}