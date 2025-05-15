## Deployment Versions

- Local environment: v1.5.6
- Development environment: v1.5.2
- Testing environment: v1.5.2
- Production environment: v1.5.5

## Changelogs

v1.5.
1. Fixed unit test files, added comments
2. Renamed files under /api/, no logic changes

v1.5.5:
1. Fixed type issues with consul ID

v1.5.4:
1. Fixed the name of the ID registered on consul

v1.5.3:
1. Fixed configuration issues in prod environment, mainly that ports cannot be the same

v1.5.2:
1. Adapted configuration files for dev environment
2. Standardized deployment of all environments to this version

v1.5.1:
1. Major changes to deployment scripts; multiple environments use the same docker-compose, differentiated only by env files

v1.5.0:
1. Added test environment configuration files and deployment scripts

v1.4.8:
1. Added support for 5 test chains and block scan support
2. Logs can be configured in text/json format
3. Fixed transaction sorting issues

v1.4.7:
1. Added integration tests for test and production environments (mainly testing cache consistency)
2. Fixed unit test failures caused by remote config
3. Assigned default native coin name for Native transactions

v1.4.6:
1. Fixed docker log path issues

v1.4.5:
1. Added configuration files required for production environment
2. Fixed real-time reading of consul remote configuration

v1.4.0-v1.4.4:
1. Application writes logs to files, defaulting to the .logs directory
2. Added integration tests (no external impact)

v1.4.0:
1. Removed redundant shadow transaction filtering code
2. Switched Redis to valkey

v1.3.5:
1. Fixed shadow transaction issues

v1.3.4:
1. Fixed issue where ankr token transactions weren't in reverse order

v1.3.3:
1. Fixed transaction sorting issues, changed to descending order
2. Resolved issues with filtering out shadow transactions

v1.3.2:
1. Removed -1 as a value from the Type field; default is 0, approve is 1

v1.3.1:
1. Implemented consul configuration center, falling back to local config files if not available
2. Fixed unit tests

v1.3.0:
1. Registered service to consul; future access will be through the kong gateway

v1.2.7:
1. Fixed filtering issues in ankr utility functions when sending requests

v1.2.6:
1. Differentiated providers by chain; ankr request parameters now differentiated by chain; ankr request parameter changed to 100 for improved speed
2. Fixed token filtering logic: when token address is provided, only returns coinType=2

v1.2.4:
1. No external impact; internal code refactoring, additional unit tests, etc.

v1.2.3:
1. Added balance field. The balance field represents values without precision handling, while the amount field represents values with precision handling.

v1.2.2:
1. The returned amount is the value after division by decimals

v1.2.1:
1. Fixed 500 error issues again

v1.2.0:
1. Changed timeout error response code to 200 instead of 500, as business-level error codes are already in place

v1.1.8:
1. Changed timeout duration to 90 seconds

v1.1.6:
1. Code refactoring, should not affect external functionality

v1.1.5:
1. Standardized the state field: 1 for success, 0 for failure

v1.1.4:
1. Optimized response speed, significantly reducing time spent on Redis operations

v1.1.3:
1. No impact on external interfaces
2. Cache keys now use chain name instead of chain ID

v1.1.2:
1. Added serverChainName field
2. chainName and tokenAddress parameters are now case-insensitive

v1.1.1:
1. Using tx type = 2 to represent internal transactions, rather than tx type = -2

v1.1.0:
1. Added tokenAddress=native parameter to query only native transactions
2. Token transfer transactions now include parameters such as gas limit, gas used, etc.

v1.0.2:
1. Gas limit, gas price, etc. now use string type instead of int