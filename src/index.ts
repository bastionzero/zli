#!/usr/bin/env node

import { CliDriver } from './cli-driver';

// console.log('thoum: 1 cup cloves garlic, 2 teaspoons salt, 1/4 cup lemon juice, 1/4 cup water, 3 cups neutral oil');
const driver = new CliDriver();
driver.start();