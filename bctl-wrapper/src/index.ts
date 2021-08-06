const { v4: uuidv4 } = require('uuid');
const spawnSync = require('child_process').spawnSync;
const spawn = require('child_process').spawn;

// First extract the args
const kubeArgs = process.argv.splice(2);
const kubeArgsString = kubeArgs.join(' ');

// Then get the token 
const getTokenProcess = spawnSync('zli', ['--configName', 'dev', 'get-kube-token', '-s']);
const token = getTokenProcess.stdout.toString('utf8');

// Now generate a log id
const logId = uuidv4();

// Now build our token
const formattedToken = `${token}bctl ${kubeArgsString}++++${logId}`;

spawn('kubectl', kubeArgs, { stdio: [process.stdin, process.stdout, process.stderr] });