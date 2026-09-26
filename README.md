# ConverGO
simple decentralized DVCS based on CvRDTs

## Run example Cluster Configuration (run also Unit tests)

**make dock-test N=3 TEST_DIR=three_peer_convergence** \
**make dock-test N=2 TEST_DIR=pairing_convergence** \
**make dock-test N=2 TEST_DIR=complex_convergence** \
**make dock-test N=3 TEST_DIR=peer_left_convergence** 

## Run example Cluster Configuration (no Unit tests)

**make dock-test-dev N=3 TEST_DIR=three_peer_convergence** \
**make dock-test-dev N=2 TEST_DIR=pairing_convergence** \
**make dock-test-dev N=2 TEST_DIR=complex_convergence** \
**make dock-test-dev N=3 TEST_DIR=peer_left_convergence** 

## Run a single node in interactive mode (run also Unit tests)

**make dock-test-int NAME_INT=envname**

## Run a single node in interactive mode (no Unit tests)

**make dock-test-dev-int NAME_INT=envname**


# Dependencies

- GNU Make 4.4.1
- Docker 20.10.23