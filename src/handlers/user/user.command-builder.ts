import yargs from "yargs";

export function userCmdBuilder(yargs: yargs.Argv<{}>) :
yargs.Argv<
{add: boolean} &
{delete: boolean} &
{idpEmail: string} &
{policyName: string} &
{json: boolean}
> {
    return yargs
        .option(
            'add',
            {
                type: 'boolean',
                demandOption: false,
                alias: 'a',
                implies: ['idpEmail', 'policyName']
            }
        )
        .option(
            'delete',
            {
                type: 'boolean',
                demandOption: false,
                alias: 'd',
                implies: ['idpEmail', 'policyName']
            }
        )
        .conflicts('add', 'delete')
        .positional('idpEmail',
            {
                type: 'string',
                default: null,
                demandOption: false,
            }
        )
        .positional('policyName',
            {
                type: 'string',
                default: null,
                demandOption: false,
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
        .example('user --json', 'List all users, output as json, pipeable')
        .example('user --add test@test.com test-cluster', 'Adds the test@test.com IDP user to test-cluster policy')
        .example('user -d test@test.com test-cluster', 'Removes the test@test.com IDP user from test-cluster policy');
}