import chalk from "chalk";
import { TargetType } from "./types";
import { last } from 'lodash'

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

export const targetStringExample: string = '[targetUser@]<targetId>:<targetPath>';
export const targetStringExampleNoPath : string = '[targetUser@]<targetId>'

export function parseTargetType(targetType: string) : TargetType
{
    const targetTypePattern = /^(ssm|ssh)/i; // case insensitive check for ssm or ssh

    if(! targetTypePattern.test(targetType))
        return undefined;

    return <TargetType> targetType.toUpperCase();
}

export function parseTargetString(targetString: string) : parsedTargetString
{
    // add a closing closing if it does not exist
    if(last(targetString) !== ':')
        targetString = targetString + ':';

    // case sensitive check for [targetUser@]<targetId>:[targetPath]
    const pattern = /^([a-z_]([a-z0-9_-]{0,31}|[a-z0-9_-]{0,30})@)?(([0-9A-Fa-f]{8}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{12})|([a-zA-Z0-9_.-]{0,255})):/;

    if(! pattern.test(targetString))
        return undefined;

    let result : parsedTargetString = {
        targetUser: undefined,
        targetId: undefined,
        targetName: undefined,
        targetPath: undefined
    };

    let atSignSplit = targetString.split('@', 2);

    // if targetUser@ is present, extract username
    if(atSignSplit.length == 2)
    {
        result.targetUser = atSignSplit[0];
        atSignSplit = atSignSplit.slice(1);
    }
    
    // extract targetId and maybe targetPath
    const colonSplit = atSignSplit[0].split(':', 2);
    const targetSomething = colonSplit[0];

    // test if targetSomething is GUID
    const guidPattern = /^[0-9A-Fa-f]{8}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{12}$/;
    if(guidPattern.test(targetSomething))
        result.targetId = targetSomething;
    else
        result.targetName = targetSomething;

    if(colonSplit[1] !== '')
        result.targetPath = colonSplit[1];

    return result;
}

export interface parsedTargetString
{
    targetUser: string;
    targetId: string;
    targetName: string;
    targetPath: string;
}

export function checkTargetTypeAndStringPair(targetType: TargetType, targetString: parsedTargetString) : boolean
{
    if(targetType === TargetType.SSH && targetString.targetUser)
    {
        thoumWarn('Cannot specify targetUser for SSH connections');
        thoumWarn('Please try your previous command without the targetUser');
        thoumWarn('Target string for SSH: targetId[:path]');
        return false;
    }

    if(targetType === TargetType.SSM && ! targetString.targetUser)
    {
        thoumWarn('Must specify targetUser for SSM connections');
        thoumWarn('Target string for SSM: targetUser@targetId[:path]');
        return false;
    }

    return true;
}