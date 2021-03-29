import { TargetType } from './types';
import { max } from 'lodash';
import { EnvironmentDetails } from './http.service/http.service.types';
import Table from 'cli-table3';

// case insensitive substring search, 'find targetString in searchString'
export function findSubstring(targetString: string, searchString: string) : boolean
{
    return searchString.toLowerCase().indexOf(targetString.toLowerCase()) !== -1;
}

export const targetStringExample: string = '[targetUser@]<targetId | targetName>:<targetPath>';
export const targetStringExampleNoPath : string = '[targetUser@]<targetId | targetName>';

export function parseTargetType(targetType: string) : TargetType
{
    const targetTypePattern = /^(ssm|ssh|dynamic)$/i; // case insensitive check for targetType

    if(! targetTypePattern.test(targetType))
        return undefined;

    return <TargetType> targetType.toUpperCase();
}

export function parseTargetString(targetTypeString: string , targetString: string) : parsedTargetString
{
    const targetType = parseTargetType(targetTypeString);

    // case sensitive check for [targetUser@]<targetId | targetName>[:targetPath]
    const pattern = /^([a-z_]([a-z0-9_-]{0,31}|[a-z0-9_-]{0,30}\$)@)?(([0-9A-Fa-f]{8}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{12})|([a-zA-Z0-9_.-]{1,255}))(:{1}|$)/;

    if(! pattern.test(targetString))
        return undefined;

    const result : parsedTargetString = {
        type: targetType,
        user: undefined,
        id: undefined,
        name: undefined,
        path: undefined
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
    const guidPattern = /^[0-9A-Fa-f]{8}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{12}$/;
    if(guidPattern.test(targetSomething))
        result.id = targetSomething;
    else
        result.name = targetSomething;

    if(colonSplit[1] !== '')
        result.path = colonSplit[1];

    return result;
}

export interface parsedTargetString
{
    type: TargetType;
    user: string;
    id: string;
    name: string;
    path: string;
}

export function checkTargetTypeAndStringPair(parsedTarget: parsedTargetString) : boolean
{
    if(parsedTarget.type === TargetType.SSH && parsedTarget.user)
        return false;

    if((parsedTarget.type === TargetType.SSM || parsedTarget.type === TargetType.DYNAMIC) && ! parsedTarget.user)
        return false;

    return true;
}

export interface TargetSummary
{
    id: string;
    name: string;
    environmentId: string;
    type: TargetType;
}

export function getTableOfTargets(targets: TargetSummary[], envs: EnvironmentDetails[], showGuid: boolean = false) : string
{
    const targetNameLength = max(targets.map(t => t.name.length).concat(16)); // if max is 0 then use 16 as width
    const envNameLength = max(envs.map(e => e.name.length).concat(16));       // same same

    let header: string[] = ['Type', 'Name', 'Environment'];
    let columnWidths = [10, targetNameLength + 2, envNameLength + 2]

    if(showGuid) 
    {
        header.push('Id');
        columnWidths.push(38);
    }

    // ref: https://github.com/cli-table/cli-table3
    var table = new Table({ head: header, colWidths: columnWidths });

    targets.forEach(target => {
            let row = [target.type, target.name, envs.filter(e => e.id == target.environmentId).pop().name];
            if(showGuid) row.push(target.id);
            table.push(row);
        }
    );

    return table.toString();
}