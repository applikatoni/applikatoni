toni is the CLI for Applikatoni.

Usage:

        toni [-c=<commit SHA>] [-b=<commit branch>] -t <target> -m <comment>

Arguments:

        REQUIRED:

        -t              Target of the deployment

        -m              The deployment comment

        OPTIONAL:

        -c      The deployment commit SHA
                        (if unspecified toni uses the current git HEAD)

        -b      The branch of the commit SHA
                        (if unspecified toni uses the current git HEAD)

Configuration:

When starting up, toni tries to read the ".toni.yml" configuration file in the
current working directoy. The configuration MUST specify the HOST, APPLICATION,
API TOKEN and the STAGES of deployments.

An example ".toni.yml" looks like this:

        host: http://toni.shippingcompany.com
        application: shippingcompany-main-application
        api_token: 4fdd575f-FOOO-BAAR-af1e-ce3e9f75367d
        stages:
          production:
                - CHECK_CONNECTION
                - PRE_DEPLOYMENT
                - CODE_DEPLOYMENT
                - POST_DEPLOYMENT
          staging:
                - CHECK_CONNECTION
                - PRE_DEPLOYMENT
                - CODE_DEPLOYMENT
                - POST_DEPLOYMENT
