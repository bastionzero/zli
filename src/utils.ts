import { TargetType } from "./types";

// case insensitive substring search, "find targetString in searchString"
export function findSubstring(targetString: string, searchString: string) : boolean
{
    return searchString.toLowerCase().indexOf(targetString.toLowerCase()) !== -1;
}

export const targetStringExample: string = '[targetUser@]<ssm|ssh>:<targetId>:<targetPath>';
export const targetStringExampleNoPath : string = '[targetUser@]<ssm|ssh>:<targetId>'

export function parseTargetString(targetString: string) : parsedTargetString
{
    // [targetUser@]<ssm|ssh>:<targetId>:[targetPath]
    const pattern = /^([a-z_]([a-z0-9_-]{0,31}|[a-z0-9_-]{0,30})@)?(ssm|ssh):([0-9A-Fa-f]{8}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{12}):/;

    if(! pattern.test(targetString))
        return undefined;

    let result : parsedTargetString = {
        targetUser: undefined,
        targetType: undefined,
        targetId: undefined,
        targetPath: undefined
    };

    let atSignSplit = targetString.split('@', 2);

    if(atSignSplit.length == 2)
    {
        result.targetUser = atSignSplit[0];
        atSignSplit = atSignSplit.slice(1);
    }
    
    const colonSplit = atSignSplit[0].split(':', 3);
    result.targetType = <TargetType> colonSplit[0].toUpperCase();
    result.targetId = colonSplit[1];
    result.targetPath = colonSplit[2];

    return result;
}

export interface parsedTargetString
{
    targetUser: string;
    targetType: TargetType;
    targetId: string;
    targetPath: string;
}