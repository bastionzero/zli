export function policyCmdBuilder (yargs : any, policyTypeChoices : string []) {
    yargs
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