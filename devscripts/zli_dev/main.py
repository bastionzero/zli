import click

from zli_dev.scripts.getToken import getToken

@click.command()
@click.option('--get-token', 'get_token', default=False, is_flag=True, help='Get Id Token and Session Id from zli dev config json')
@click.option('--zli-path', 'zli_path', default=None, help='Custom path to use for zli executable')
@click.option('--configName', 'config_name', default="prod", help='Config file to use [prod, stage, dev]')
def main(get_token, zli_path, config_name):
    if (get_token):
        getToken(zli_path, config_name)

if __name__ == "__main__":
    main()