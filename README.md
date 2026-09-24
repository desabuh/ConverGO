# ConverGO
simple decentralized DVCS based on CvRDTs

## Run example Cluster Configuration (slower but run also Unit tests)

**make dock-test N=3 TEST_DIR=three_peer_convergence** \
**make dock-test N=2 TEST_DIR=pairing_convergence** \
**make dock-test N=2 TEST_DIR=complex_convergence** \
**make dock-test N=3 TEST_DIR=peer_left_convergence** 

## Run example Cluster Configuration (faster without Unit tests)

**make dock-test-dev N=3 TEST_DIR=three_peer_convergence** \
**make dock-test-dev N=2 TEST_DIR=pairing_convergence** \
**make dock-test-dev N=2 TEST_DIR=complex_convergence** \
**make dock-test-dev N=3 TEST_DIR=peer_left_convergence** 

# Dependencies

- GNU Make 4.4.1
- Docker 20.10.23