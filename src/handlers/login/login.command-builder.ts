import yargs from 'yargs';

export type loginArgs = {mfa: string}

export function loginCmdBuilder (yargs : yargs.Argv<{}>) : yargs.Argv<loginArgs>
{
    return yargs
        .option(
            'mfa',
            {
                type: 'string',
                demandOption: false,
                alias: 'm'
            }
        )
        .example('$0 login --mfa 123456', 'Login and enter MFA');
}