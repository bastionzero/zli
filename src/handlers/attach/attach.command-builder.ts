import yargs from "yargs";

export type attachArgs = {connectionId : string}

export function attachCmdBuilder(yargs : yargs.Argv<{}>) : yargs.Argv<attachArgs> {
    return yargs
        .positional('connectionId', {
            type: 'string',
        })
        .example('attach d5b264c7-534c-4184-a4e4-3703489cb917', 'attach example, unique connection id');
}