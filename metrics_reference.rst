Metrics Reference
=================

Prometheus Metrics
------------------

The following metrics are currently exported for consumption by Prometheus.

+----------------------------------+-----------+------------------------------------------------------------+--------------------+
| Name                             | Type      | Description                                                | Labels             |
+==================================+===========+============================================================+====================+
| config_periodic_refresh_duration | histogram | The config periodic refresh duration.                      |                    |
+----------------------------------+-----------+------------------------------------------------------------+--------------------+
| config_refresh_duration          | histogram | The config refresh duration.                               |                    |
+----------------------------------+-----------+------------------------------------------------------------+--------------------+
| config_service_refresh_duration  | histogram | The config refresh duration.                               |                    |
+----------------------------------+-----------+------------------------------------------------------------+--------------------+
| http_count                       | counter   | The number of http calls.                                  |                    |
+----------------------------------+-----------+------------------------------------------------------------+--------------------+
| http_duration                    | histogram | The http call duration.                                    |                    |
+----------------------------------+-----------+------------------------------------------------------------+--------------------+
| proposal_count                   | counter   | The number of proposal.                                    |                    |
+----------------------------------+-----------+------------------------------------------------------------+--------------------+
| proposal_duration                | histogram | The proposal duration.                                     |                    |
+----------------------------------+-----------+------------------------------------------------------------+--------------------+
| proposal_error_count             | counter   | The number of failed proposal.                             |                    |
+----------------------------------+-----------+------------------------------------------------------------+--------------------+
| transaction_retry                | counter   | The number of transaction retry.                           |                    |
+----------------------------------+-----------+------------------------------------------------------------+--------------------+


StatsD Metrics
--------------

The following metrics are currently emitted for consumption by StatsD. The
``%{variable_name}`` nomenclature represents segments that vary based on
context.

For example, ``%{channel}`` will be replaced with the name of the channel
associated with the metric.

+----------------------------------+-----------+------------------------------------------------------------+
| Bucket                           | Type      | Description                                                |
+==================================+===========+============================================================+
| config.periodic_refresh_duration | histogram | The config periodic refresh duration.                      |
+----------------------------------+-----------+------------------------------------------------------------+
| config.refresh_duration          | histogram | The config refresh duration.                               |
+----------------------------------+-----------+------------------------------------------------------------+
| config_service.refresh_duration  | histogram | The config refresh duration.                               |
+----------------------------------+-----------+------------------------------------------------------------+
| http.count                       | counter   | The number of http calls.                                  |
+----------------------------------+-----------+------------------------------------------------------------+
| http.duration                    | histogram | The http call duration.                                    |
+----------------------------------+-----------+------------------------------------------------------------+
| proposal.count                   | counter   | The number of proposal.                                    |
+----------------------------------+-----------+------------------------------------------------------------+
| proposal.duration                | histogram | The proposal duration.                                     |
+----------------------------------+-----------+------------------------------------------------------------+
| proposal.error_count             | counter   | The number of failed proposal.                             |
+----------------------------------+-----------+------------------------------------------------------------+
| transaction.retry                | counter   | The number of transaction retry.                           |
+----------------------------------+-----------+------------------------------------------------------------+


.. Licensed under Creative Commons Attribution 4.0 International License
   https://creativecommons.org/licenses/by/4.0/
