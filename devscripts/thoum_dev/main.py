import click

from thoum_dev.scripts.getToken import getToken

@click.command()
@click.option('--get-token', 'get_token', default=False, is_flag=True, help='Get Id Token and Session Id from thoum dev config json')
@click.option('--thoum-path', 'thoum_path', default=None, help='Custom path to use for thoum executable')
@click.option('--configName', 'config_name', default="prod", help='Config file to use [prod, stage, dev]')
def main(get_token, thoum_path, config_name):
    if (get_token):
        getToken(thoum_path, config_name)

if __name__ == "__main__":
    main()