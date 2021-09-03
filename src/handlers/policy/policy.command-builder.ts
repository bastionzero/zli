import yargs from "yargs";

export function policyCmdBuilder (yargs : yargs.Argv<{}>, policyTypeChoices : string []) :
yargs.Argv<{type: string;} & {json: boolean;}> {
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
        .example('policy --json', 'List all policies, output as json, pipeable');
}