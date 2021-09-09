import yargs from 'yargs';

type attachArgs = {connectionId : string}

export function attachCmdBuilder(yargs : yargs.Argv<{}>) : yargs.Argv<attachArgs> {
    return yargs
        .positional('connectionId', {
            type: 'string',
        })
        .example('$0 attach d5b264c7-534c-4184-a4e4-3703489cb917', 'attach example, unique connection id');
}