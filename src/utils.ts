import { max, filter, concat, map } from 'lodash';
import Table from 'cli-table3';
import { Logger } from './services/logger/logger.service';
import { cleanExit } from './handlers/clean-exit.handler';
import { ApiKeyDetails } from './services/api-key/api-key.types';
import { TargetType, TargetStatus, ParsedTargetString, TargetSummary } from './services/common.types';
import { ConnectionDetails } from './services/connection/connection.types';
import { EnvironmentDetails } from './services/environment/environment.types';
import { GroupSummary } from './services/groups/groups.types';
import { PolicyType, PolicySummary, SubjectType, KubernetesPolicyContext, TargetConnectContext } from './services/policy/policy.types';
import { UserSummary } from './services/user/user.types';

// case insensitive substring search, 'find targetString in searchString'
export function findSubstring(targetString: string, searchString: string) : boolean
{
    return searchString.toLowerCase().indexOf(targetString.toLowerCase()) !== -1;
}

export const targetStringExample : string = '[targetUser@]<targetId-or-targetName>';

export function parseTargetType(targetType: string) : TargetType
{
    const targetTypePattern = /^(ssm|ssh|dynamic|cluster)$/i; // case insensitive check for targetType

    if(! targetTypePattern.test(targetType))
        return undefined;

    return <TargetType> targetType.toUpperCase();
}

export function parsePolicyType(policyType: string) : PolicyType
{
    const policyTypePattern = /^(targetconnect|organizationcontrols|sessionrecording|kubernetestunnel)$/i; // case insensitive check for policyType

    if(! policyTypePattern.test(policyType))
        return undefined;

    switch (policyType.toLowerCase()) {
    case PolicyType.KubernetesTunnel.toLowerCase():
        return PolicyType.KubernetesTunnel;
    case PolicyType.OrganizationControls.toLowerCase():
        return PolicyType.OrganizationControls;
    case PolicyType.SessionRecording.toLowerCase():
        return PolicyType.SessionRecording;
    case PolicyType.TargetConnect.toLowerCase():
        return PolicyType.TargetConnect;
    default:
        return undefined;
    }
}

export function parseTargetStatus(targetStatus: string) : TargetStatus {
    switch (targetStatus.toLowerCase()) {
    case TargetStatus.NotActivated.toLowerCase():
        return TargetStatus.NotActivated;
    case TargetStatus.Offline.toLowerCase():
        return TargetStatus.Offline;
    case TargetStatus.Online.toLowerCase():
        return TargetStatus.Online;
    case TargetStatus.Terminated.toLowerCase():
        return TargetStatus.Terminated;
    case TargetStatus.Error.toLowerCase():
        return TargetStatus.Error;
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
    // The following constant numbers are set specifically to conform with the specified 80/132 cols term size - do not change
    const targetNameLength = max(targets.map(t => t.name.length)) + 2 || 16; // || 16 here means that when there are no targets default the length to 16
    const envNameLength = max(envs.map(e => e.name.length)) + 2 < 16 ? 16 : max(envs.map(e => e.name.length));

    const header: string[] = ['Type', 'Name', 'Environment'];
    const columnWidths = [];
    if (!showDetail) {
        columnWidths.push(9);
        columnWidths.push(targetNameLength > 44 ? 44 : targetNameLength);
        columnWidths.push(envNameLength > 47 ? 47 : envNameLength);
    } else {
        columnWidths.push(9);
        columnWidths.push(targetNameLength > 32 ? 32 : targetNameLength);
        columnWidths.push(envNameLength > 31 ? 31 : envNameLength);
    }

    if(showGuid)
    {
        header.push('Id');
        columnWidths.push(38);
    }

    if(showDetail)
    {
        header.push('Agent Version', 'Status', 'Target Users');
        columnWidths.push(15, 9, 29);
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
            row.push(map(target.targetUsers).join(', \n') || 'N/A'); // targetUsers are undefined for now for non-cluster targets
        }

        table.push(row);
    }
    );

    return table.toString();
}

export function getTableOfConnections(connections: ConnectionDetails[], allTargets: TargetSummary[]) : string
{
    const connIdLength = max(connections.map(c => c.id.length).concat(36));
    const targetUserLength = max(connections.map(c => c.userName.length).concat(16));
    const targetNameLength = max(allTargets.map(t => t.name.length).concat(16));
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

export function getTableOfUsers(users: UserSummary[]) : string
{
    const nameLength = max(users.map(u => u.fullName.length).concat(16));
    const emailLength = max(users.map(u => u.email.length).concat(36));
    const header: string[] = ['Name', 'Email', 'Role', 'Last Login'];
    const columnWidths = [nameLength + 2, emailLength + 2, 7, 20];

    const table = new Table({ head: header, colWidths: columnWidths });
    const dateOptions = {year: '2-digit', month: 'numeric', day: 'numeric', hour:'numeric', minute:'numeric', hour12: true};
    users.forEach(u => {
        const row = [u.fullName, u.email, u.isAdmin ? 'Admin' : 'User', new Date(u.lastLogin).toLocaleString('en-US', dateOptions as any)];
        table.push(row);
    });

    return table.toString();
}

export function getTableOfGroups(groups: GroupSummary[]) : string
{
    const nameLength = max(groups.map(g => g.name.length).concat(16));
    const header: string[] = ['Group Name'];
    const columnWidths = [nameLength + 2];

    const table = new Table({ head: header, colWidths: columnWidths });
    groups.forEach(g => {
        const row = [g.name];
        table.push(row);
    });

    return table.toString();
}

export function getTableOfTargetUsers(targetUsers: string[]) : string
{
    const header: string[] = ['Allowed Target Users'];
    const nameLength = max(targetUsers.map(u => u.length).concat(16));
    // If the title's length is bigger than the longer user use that as the row length
    const rowLength = nameLength > header[0].length ? nameLength : header[0].length;
    const columnWidths = [rowLength + 2];

    const table = new Table({ head: header, colWidths: columnWidths });
    targetUsers.forEach(u => {
        const row = [u];
        table.push(row);
    });

    return table.toString();
}

export function getTableOfDescribeCluster(policies: PolicySummary[], targetUsers: string[], environmentName : string) : string {
    const header: string[] = ['Policy', 'Environment', 'Target Users'];

    const policyLength = max(policies.map(p => p.name.length).concat(16));
    const targetUserLength = max(targetUsers.map(u => u.length).concat(16));

    const columnWidths = [policyLength + 2, environmentName.length + 2, targetUserLength + 4];

    const formattedTargetUsers = targetUsers.join( ', \n');

    const table = new Table({ head: header, colWidths: columnWidths });
    policies.forEach(p => {
        const row = [p.name, environmentName, formattedTargetUsers];
        table.push(row);
    });

    return table.toString();
}

// TODO : The following functionality is very similar to the webapp, it could be abstracted to common-ts
export function getTableOfPolicies(
    policies: PolicySummary[],
    userMap: {[id: string]: UserSummary},
    apiKeyMap: {[id: string]: ApiKeyDetails},
    environmentMap: {[id: string]: EnvironmentDetails},
    targetMap : {[id: string]: string},
    groupMap : {[id: string]: GroupSummary}
) : string
{
    const header: string[] = ['Name', 'Type', 'Subject', 'Resource', 'Target Users'];
    const columnWidths = [24, 19, 26, 28, 29];

    const table = new Table({ head: header, colWidths: columnWidths });
    policies.forEach(p => {

        // Translate the policy subject ids to human readable subjects
        const groupNames : string [] = [];
        p.groups.forEach(group => {
            groupNames.push(getGroupName(group.id, groupMap));
        });
        const formattedGroups = !! groupNames.length ? 'Groups: ' + groupNames.join( ', \n') : '';

        const subjectNames : string [] = [];
        p.subjects.forEach(subject => {
            switch (subject.type) {
            case SubjectType.ApiKey:
                subjectNames.push('ApiKey:' + getApiKeyName(subject.id, apiKeyMap));
                break;
            case SubjectType.User:
                subjectNames.push(getUserName(subject.id, userMap));
                break;
            default:
                break;
            }
        });
        let formattedSubjects = subjectNames.join( ', \n');
        if (subjectNames.length > 0 && !!formattedGroups) {
            formattedSubjects += '\n';
        }
        formattedSubjects += formattedGroups;

        // Translate the resource ids to human readable resources
        // TODO : This should get extended to support other policy types as well
        let formattedResource = '';
        let formattedTargetUsers = '';
        if (p.type == PolicyType.KubernetesTunnel) {
            const kubernetesPolicyContext = p.context as KubernetesPolicyContext;
            // If this policy gets applied on some environments
            if (kubernetesPolicyContext.environments) {
                const environmentNames : string [] = [];
                Object.keys(kubernetesPolicyContext.environments).forEach(
                    envId => environmentNames.push(getEnvironmentName(envId, environmentMap))
                );
                formattedResource = 'Environments: ' + environmentNames.join( ', \n');
            } else if (kubernetesPolicyContext.clusters) { // Alternatively if this policy gets applied straight on some clusters
                const clusterNames : string [] = [];
                Object.values(kubernetesPolicyContext.clusters).forEach(
                    cluster => clusterNames.push(getTargetName(cluster.id, targetMap))
                );
                formattedResource = 'Clusters: ' + clusterNames.join( ', \n');
            }

            if (kubernetesPolicyContext.clusterUsers) {
                const clusterUsersNames : string [] = [];
                Object.values(kubernetesPolicyContext.clusterUsers).forEach(
                    clusterUser => clusterUsersNames.push(clusterUser.name)
                );
                formattedTargetUsers = 'Cluster Users: ' + clusterUsersNames.join( ', \n');
            }
        } else if (p.type == PolicyType.TargetConnect) {
            const targetAccessContext = p.context as TargetConnectContext;
            // If this policy gets applied on some environments
            if (targetAccessContext.environments && Object.keys(targetAccessContext.environments).length > 0) {
                const environmentsResourceStrings: string [] = [];
                Object.values(targetAccessContext.environments).forEach(env => {
                    environmentsResourceStrings.push(getEnvironmentName(env.id, environmentMap));
                });
                formattedResource = 'Environments: ' + environmentsResourceStrings.join( ', \n');
            } else if (targetAccessContext.targets && Object.keys(targetAccessContext.targets).length > 0) { // Alternatively if this policy gets applied straight on some targets
                const targetResourceStrings: string [] = [];
                Object.values(targetAccessContext.targets).forEach(target => {
                    targetResourceStrings.push(getTargetName(target.id, targetMap));
                });
                formattedResource = 'Targets: ' + targetResourceStrings.join( ', \n');
            }

            if (targetAccessContext.targetUsers && Object.keys(targetAccessContext.targetUsers).length > 0) {
                const targetUsersStrings: string [] = [];
                Object.values(targetAccessContext.targetUsers).forEach(tu => {
                    targetUsersStrings.push(tu.userName);
                });
                formattedTargetUsers = 'Unix Users: ' + targetUsersStrings.join( ', \n');
            }

        }

        const row = [
            p.name,
            p.type,
            formattedSubjects || 'N/A',
            formattedResource || 'N/A',
            formattedTargetUsers || 'N/A'
        ];
        table.push(row);
    });

    return table.toString();
}

function getApiKeyName(apiKeyId: string, apiKeyMap: {[id: string]: ApiKeyDetails}) : string {
    return apiKeyMap[apiKeyId]
        ? apiKeyMap[apiKeyId].name
        : 'API KEY DELETED';
}

function getUserName(userId: string, userMap: {[id: string]: UserSummary}) : string {
    return userMap[userId]
        ? userMap[userId].fullName
        : 'USER DELETED';
}

function getEnvironmentName(envId: string, environmentMap: {[id: string]: EnvironmentDetails}) : string {
    return environmentMap[envId]
        ? environmentMap[envId].name
        : 'ENVIRONMENT DELETED';
}

function getTargetName(targetId: string, targetMap: {[id: string]: string}) : string {
    return targetMap[targetId]
        ? targetMap[targetId]
        : 'TARGET DELETED';
}

function getGroupName(groupId: string, groupMap: {[id: string]: GroupSummary}) : string {
    return groupMap[groupId]
        ? groupMap[groupId].name
        : 'GROUP DELETED';
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
    zippedTargets = filter(zippedTargets, t => t.type !== TargetType.SSM || (t.status !== TargetStatus.Error && t.status !== TargetStatus.Terminated));

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