import yargs from 'yargs';

export type listConnectionsArgs = {json: boolean}

export function listConnectionsCmdBuilder(yargs: yargs.Argv<{}>) : yargs.Argv<listConnectionsArgs> {
    return yargs
        .option(
            'json',
            {
                type: 'boolean',
                default: false,
                demandOption: false,
                alias: 'j',
            }
        )
        .example('$0 lc --json', 'List all open zli connections, output as json, pipeable');
}