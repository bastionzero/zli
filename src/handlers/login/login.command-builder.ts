import { IdP } from '../../services/common.types';
import yargs from 'yargs';

export type loginArgs = {provider: string;} & {mfa: string}

export function loginCmdBuilder (yargs : yargs.Argv<{}>) : yargs.Argv<loginArgs>
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
        .example('$0 login Google', 'Login with Google')
        .example('$0 login Microsoft --mfa 123456', 'Login with Microsoft and enter MFA');
}