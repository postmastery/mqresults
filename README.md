# mqresults

Post MailerQ [results](https://www.mailerq.com/documentation/json-results) to HTTP endpoint.

The "results" queue is used by MailerQ to publish all the results: both the successful results as well as the failures. The "success" and "failures" queues are used for just the successes and just the failures. These queues receive a subset of the messages published to the "results" queue. The "retry" queue is used for transient results for deliveries that have not yet failed or succeeded, and that are going to be retried.

## Setup

### Configure MailerQ

Configuration to receive success, failed and retries in the "results" queue:

    rabbitmq-results:  results
    rabbitmq-success:
    rabbitmq-failure:
    rabbitmq-retry:    results

See also [RabbitMQ configuration](https://www.mailerq.com/documentation/rabbitmq-config).

### Install mqresults

Download mqresults for Linux using the browser from [https://postmastery.egnyte.com/dl/Wb3ib8rbh9](https://postmastery.egnyte.com/dl/Wb3ib8rbh9).

Copy mqresults to /usr/local/sbin on the server. Remove any extension and make it executable:

    chmod +x /usr/local/sbin/mqresults

Use -help to show all command line parameters:

    mqresults -help

Set the -rabbitmq-address parameter to the same URI as configured in the MailerQ configuration. The -endpoint-url and -authorization parameters are provided by Postmastery.

Test connection to RabbitMQ:

    mqresults -rabbitmq-address="amqp://guest:guest@localhost" -endpoint-url="https://hostname/your/endpoint" -authorization="your_access_key"

Wait until "Posted ... events ..." or "No messages yet" is logged. Press Ctrl+C to quit the application.

### Run as service

#### Ubuntu LTS 12-14 or RHEL/CentOS 6

Create an upstart script as /etc/init/mqresults.conf:

    description "Upstart configuration for mqresults"

    start on filesystem or runlevel [2345]
    stop on shutdown

    exec /usr/local/sbin/mqresults -rabbitmq-address="amqp://guest:guest@localhost" -endpoint-url="https://hostname/path/to/endpoint" -authorization="your_access_key"

    respawn

Reload Upstart configurations:

    initctl reload-configuration

Start mqresults as daemon:

    initctl start mqresults

#### Ubuntu LTS 16+ or RHEL/CentOS 7

Create systemd configuration as /lib/systemd/system/mqresults.service:
    
    [Unit]
    Description=Systemd configuration for mqresults

    [Service]
    ExecStart=/usr/local/sbin/mqresults -rabbitmq-address="amqp://guest:guest@localhost" -endpoint-url="https://hostname/path/to/endpoint" -authorization="your_access_key"
    Restart=on-failure

    [Install]
    WantedBy=multi-user.target

Start mqresults:

    systemctl start mqresults

View mqresults logging:

    sudo journalctl -u mqresults

