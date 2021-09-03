import yargs from "yargs";

export function sshProxyCmdBuilder(yargs: yargs.Argv<{}>) :
yargs.Argv<
{host: string} &
{user: string} &
{port: number} &
{identityFile: string}
> {
    return yargs
        .positional('host', {
            type: 'string',
        })
        .positional('user', {
            type: 'string',
        })
        .positional('port', {
            type: 'number',
        })
        .positional('identityFile', {
            type: 'string'
        });
}