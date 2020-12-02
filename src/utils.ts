import chalk from "chalk";
import { TargetType } from "./types";

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
    const targetTypePattern = /^(ssm|ssh)/i;

    if(! targetTypePattern.test(targetType))
        return undefined;

    return <TargetType> targetType.toUpperCase();
}

export function parseTargetString(targetString: string) : parsedTargetString
{
    // [targetUser@]<ssm|ssh>:<targetId>:[targetPath]
    const pattern = /^([a-z_]([a-z0-9_-]{0,31}|[a-z0-9_-]{0,30})@)?([0-9A-Fa-f]{8}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{12}):/;

    if(! pattern.test(targetString))
        return undefined;

    let result : parsedTargetString = {
        targetUser: undefined,
        targetId: undefined,
        targetPath: undefined
    };

    let atSignSplit = targetString.split('@', 2);

    if(atSignSplit.length == 2)
    {
        result.targetUser = atSignSplit[0];
        atSignSplit = atSignSplit.slice(1);
    }
    
    const colonSplit = atSignSplit[0].split(':', 2);
    result.targetId = colonSplit[0];
    result.targetPath = colonSplit[1];

    return result;
}

export interface parsedTargetString
{
    targetUser: string;
    targetId: string;
    targetPath: string;
}