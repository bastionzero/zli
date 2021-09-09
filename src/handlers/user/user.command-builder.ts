import yargs from 'yargs';

export type userArgs = {add: boolean} &
{delete: boolean} &
{idpEmail: string} &
{policyName: string} &
{json: boolean}

export function userCmdBuilder(yargs: yargs.Argv<{}>) : yargs.Argv<userArgs> {
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
        .example('$0 user --json', 'List all users, output as json, pipeable')
        .example('$0 user --add test@test.com test-cluster', 'Adds the test@test.com IDP user to test-cluster policy')
        .example('$0 user -d test@test.com test-cluster', 'Removes the test@test.com IDP user from test-cluster policy');
}