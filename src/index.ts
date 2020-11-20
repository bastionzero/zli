import { CliDriver } from "./cli-driver";
import chalk from 'chalk';
import figlet from "figlet";

// console.log('thoum: 1 cup cloves garlic, 2 teaspoons salt, 1/4 cup lemon juice, 1/4 cup water, 3 cups neutral oil');

console.log(
    chalk.magentaBright(
        figlet.textSync('clunk80 cli', { horizontalLayout: 'full' })
    )
);

var driver = new CliDriver();
driver.start();