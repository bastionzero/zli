import chalk from "chalk";
import { TargetType } from "./types";
import { last } from 'lodash'
import { EnvironmentDetails } from "./http.service/http.service.types";
import Table from 'cli-table3';

export function thoumMessage(message: string): void
{
    console.log(chalk.magenta(`thoum >>> ${message}`));
}

export function thoumWarn(message: string): void
{
    console.log(chalk.yellowBright(`thoum >>> ${message}`));
}

export function thoumError(message: string): void
{
    console.log(chalk.red(`thoum >>> ${message}`));
}

// case insensitive substring search, "find targetString in searchString"
export function findSubstring(targetString: string, searchString: string) : boolean
{
    return searchString.toLowerCase().indexOf(targetString.toLowerCase()) !== -1;
}

export const targetStringExample: string = '[targetUser@]<targetId | targetName>:<targetPath>';
export const targetStringExampleNoPath : string = '[targetUser@]<targetId | targetName>'

export function parseTargetType(targetType: string) : TargetType
{
    const targetTypePattern = /^(ssm|ssh)/i; // case insensitive check for ssm or ssh

    if(! targetTypePattern.test(targetType))
        return undefined;

    return <TargetType> targetType.toUpperCase();
}

export function parseTargetString(targetTypeString: string , targetString: string) : parsedTargetString
{
    const targetType = parseTargetType(targetTypeString);

    // case sensitive check for [targetUser@]<targetId | targetName>[:targetPath]
    const pattern = /^([a-z_]([a-z0-9_-]{0,31}|[a-z0-9_-]{0,30})@)?(([0-9A-Fa-f]{8}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{12})|([a-zA-Z0-9_.-]{0,255}))(:{1}|$)/;

    if(! pattern.test(targetString))
        return undefined;

    let result : parsedTargetString = {
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
    {
        thoumWarn('Cannot specify targetUser for SSH connections');
        thoumWarn('Please try your previous command without the targetUser');
        thoumWarn('Target string for SSH: targetId[:path]');
        return false;
    }

    if(parsedTarget.type === TargetType.SSM && ! parsedTarget.user)
    {
        thoumWarn('Must specify targetUser for SSM connections');
        thoumWarn('Target string for SSM: targetUser@targetId[:path]');
        return false;
    }

    return true;
}

export interface TargetSummary
{
    id: string;
    name: string;
    environmentId: string;
    type: TargetType;
}

export function getTableOfTargets(targets: TargetSummary[], envs: EnvironmentDetails[]) : string
{
    // ref: https://github.com/cli-table/cli-table3
    var table = new Table({
        head: ['Type', 'Name', 'Environment', 'Id']
    , colWidths: [6, 16, 16, 38]
    });

    targets.forEach(target => table.push([target.type, target.name, envs.filter(e => e.id == target.environmentId).pop().name, target.id]));

    return table.toString();
}