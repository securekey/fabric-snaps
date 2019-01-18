Metrics Reference
=================

Prometheus Metrics
------------------

The following metrics are currently exported for consumption by Prometheus.

+---------------------------------------+-----------+------------------------------------------------------------+--------------------+
| Name                                  | Type      | Description                                                | Labels             |
+=======================================+===========+============================================================+====================+
| snap_config_periodic_refresh_duration | histogram | The config periodic refresh duration.                      |                    |
+---------------------------------------+-----------+------------------------------------------------------------+--------------------+
| snap_config_refresh_duration          | histogram | The config refresh duration.                               |                    |
+---------------------------------------+-----------+------------------------------------------------------------+--------------------+
| snap_config_service_refresh_duration  | histogram | The config refresh duration.                               |                    |
+---------------------------------------+-----------+------------------------------------------------------------+--------------------+
| snap_http_count                       | counter   | The number of http calls.                                  |                    |
+---------------------------------------+-----------+------------------------------------------------------------+--------------------+
| snap_http_duration                    | histogram | The http call duration.                                    |                    |
+---------------------------------------+-----------+------------------------------------------------------------+--------------------+
| snap_proposal_count                   | counter   | The number of proposal.                                    |                    |
+---------------------------------------+-----------+------------------------------------------------------------+--------------------+
| snap_proposal_duration                | histogram | The proposal duration.                                     |                    |
+---------------------------------------+-----------+------------------------------------------------------------+--------------------+
| snap_proposal_error_count             | counter   | The number of failed proposal.                             |                    |
+---------------------------------------+-----------+------------------------------------------------------------+--------------------+
| snap_txn_retry                        | counter   | The number of transaction retry.                           |                    |
+---------------------------------------+-----------+------------------------------------------------------------+--------------------+


StatsD Metrics
--------------

The following metrics are currently emitted for consumption by StatsD. The
``%{variable_name}`` nomenclature represents segments that vary based on
context.

For example, ``%{channel}`` will be replaced with the name of the channel
associated with the metric.

+---------------------------------------+-----------+------------------------------------------------------------+
| Bucket                                | Type      | Description                                                |
+=======================================+===========+============================================================+
| snap.config.periodic_refresh_duration | histogram | The config periodic refresh duration.                      |
+---------------------------------------+-----------+------------------------------------------------------------+
| snap.config.refresh_duration          | histogram | The config refresh duration.                               |
+---------------------------------------+-----------+------------------------------------------------------------+
| snap.config_service.refresh_duration  | histogram | The config refresh duration.                               |
+---------------------------------------+-----------+------------------------------------------------------------+
| snap.http.count                       | counter   | The number of http calls.                                  |
+---------------------------------------+-----------+------------------------------------------------------------+
| snap.http.duration                    | histogram | The http call duration.                                    |
+---------------------------------------+-----------+------------------------------------------------------------+
| snap.proposal_count                   | counter   | The number of proposal.                                    |
+---------------------------------------+-----------+------------------------------------------------------------+
| snap.proposal_duration                | histogram | The proposal duration.                                     |
+---------------------------------------+-----------+------------------------------------------------------------+
| snap.proposal_error_count             | counter   | The number of failed proposal.                             |
+---------------------------------------+-----------+------------------------------------------------------------+
| snap.txn.retry                        | counter   | The number of transaction retry.                           |
+---------------------------------------+-----------+------------------------------------------------------------+


.. Licensed under Creative Commons Attribution 4.0 International License
   https://creativecommons.org/licenses/by/4.0/
