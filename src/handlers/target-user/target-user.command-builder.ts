import yargs from 'yargs';

export type targetUserArgs = {add: boolean} &
{delete: boolean} &
{user: string} &
{policyName: string} &
{json: boolean}

export function targetUserCmdBuilder(yargs: yargs.Argv<{}>) :
yargs.Argv<targetUserArgs> {
    return yargs
        .option(
            'add',
            {
                type: 'boolean',
                demandOption: false,
                alias: 'a',
                implies: ['user', 'policyName']
            }
        )
        .option(
            'delete',
            {
                type: 'boolean',
                demandOption: false,
                alias: 'd',
                implies: ['user', 'policyName']
            }
        )
        .conflicts('add', 'delete')
        .positional('user',
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
                demandOption: true,
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
        .example('$0 targetUser --json', 'List all target users, output as json, pipeable')
        .example('$0 targetUser --add cool-policy centos', 'Adds the centos user to cool-policy')
        .example('$0 targetUser -d test-cluster admin', 'Removes the admin RBAC Role to the test-cluster policy');
}