import yargs from 'yargs';

type closeConnectionArgs = {connectionId : string;} & {all : boolean}

export function closeConnectionCmdBuilder(yargs : yargs.Argv<{}>) : yargs.Argv<closeConnectionArgs> {
    return yargs
        .positional('connectionId', {
            type: 'string',
        })
        .option(
            'all',
            {
                type: 'boolean',
                default: false,
                demandOption: false,
                alias: 'a'
            }
        )
        .example('$0 close d5b264c7-534c-4184-a4e4-3703489cb917', 'close example, unique connection id')
        .example('$0 close all', 'close all connections in cli-space');
}