from setuptools import setup, find_packages
import os

with open('requirements.txt', 'rb') as f:
    install_requires = f.read().decode('utf-8').split('\n')

setup(
    name='thoum-dev',
    version=1.0,
    description="thoum dev scripts",
    author='Sid Premkumar',
    author_email='sid@commonwealthcrypto.com',
    install_requires=install_requires,
    packages=find_packages("thoum_dev"),
    entry_points={
        'console_scripts': [
            "thoum-dev=thoum_dev.main:main",
        ],
    },
)