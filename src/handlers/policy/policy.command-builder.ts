import yargs from 'yargs';

export type policyArgs = {type: string} & {json: boolean}

export function policyCmdBuilder (yargs : yargs.Argv<{}>, policyTypeChoices : string []) : yargs.Argv<policyArgs> {
    return yargs
        .option(
            'type',
            {
                type: 'string',
                choices: policyTypeChoices,
                alias: 't',
                demandOption: false
            }
        )
        .option(
            'json',
            {
                type: 'boolean',
                default: false,
                demandOption: false,
                alias: 'j',
            }
        )
        .example('$0 policy --json', 'List all policies, output as json, pipeable');
}