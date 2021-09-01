import yargs from 'yargs';
import { IdP } from '../../types';

export function loginCmdBuilder (yargs : yargs.Argv<{}>) :
yargs.Argv<{
    provider: string;
} & {
    mfa: string;
}>
{
    return yargs
        .positional('provider', {
            type: 'string',
            choices: [IdP.Google, IdP.Microsoft]
        })
        .option(
            'mfa',
            {
                type: 'string',
                demandOption: false,
                alias: 'm'
            }
        )
        .example('login Google', 'Login with Google')
        .example('login Microsoft --mfa 123456', 'Login with Microsoft and enter MFA');
}