from setuptools import setup, find_packages

with open('requirements.txt', 'rb') as f:
    install_requires = f.read().decode('utf-8').split('\n')

setup(
    name='bctl',
    version=1.0,
    description="Bctl CLI",
    author='Sid Premkumar',
    author_email='sid@bastionzero.com',
    install_requires=install_requires,
    packages=find_packages('scripts'),
    entry_points={
        'console_scripts': [
            'bctl=bctl.main:main',
        ],
    },
)