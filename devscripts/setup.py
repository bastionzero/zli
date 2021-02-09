from setuptools import setup, find_packages
import os

with open('requirements.txt', 'rb') as f:
    install_requires = f.read().decode('utf-8').split('\n')

setup(
    name='zli-dev',
    version=1.0,
    description="zli dev scripts",
    author='Sid Premkumar',
    author_email='sid@commonwealthcrypto.com',
    install_requires=install_requires,
    packages=find_packages("zli_dev"),
    entry_points={
        'console_scripts': [
            "zli-dev=zli_dev.main:main",
        ],
    },
)