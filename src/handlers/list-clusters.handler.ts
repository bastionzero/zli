import {
    findSubstring,
    parseClusterStatus,
    getTableOfClusters
} from '../utils';
import { Logger } from '../logger.service/logger';
import { EnvironmentDetails } from '../http.service/http.service.types';
import { cleanExit } from './clean-exit.handler';
import { ClusterSummary, KubeClusterStatus } from '../types';
import _ from 'lodash';


export async function listClustersHandler(
    logger: Logger,
    argv: any,
    clusterTargets: Promise<ClusterSummary[]>) {
    // await 
    var clusters = await clusterTargets;

    // filter targets by name/alias
    if(!! argv.name) {
        clusters = clusters.filter(t => findSubstring(argv.name, t.name));
    }

    if(!! argv.status) {
        const statusArray: string[] = argv.status;

        if(statusArray.length < 1) {
            logger.warn('Status filter flag passed with no arguments, please indicate at least one status');
            await cleanExit(1, logger);
        }

        let kubeStatusFilter: KubeClusterStatus[] = _.map(statusArray, (s: string) => parseClusterStatus(s)).filter(s => s); // filters out undefined
        kubeStatusFilter = _.uniq(kubeStatusFilter);

        if(kubeStatusFilter.length < 1) {
            logger.warn('Status filter flag passed with no valid arguments, please indicate at least one valid status');
            await cleanExit(1, logger);
        }

        clusters = clusters.filter(t => _.includes(kubeStatusFilter, t.status));
    }

    if(!! argv.json) {
        // json output
        console.log(JSON.stringify(clusters));
    } else {
        // regular table output
        // We OR the detail and status flags since we want to show the details in both cases
        const tableString = getTableOfClusters(clusters, !! argv.detail || !! argv.showId);
        console.log(tableString);
    }
    await cleanExit(0, logger);
}