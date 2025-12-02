Lab 3: Chord
Implement a (secure) simple distributed storage based on Chord

 

1. Introduction
In this project, you will implement Chord. To do so, you must read the paper describing the Chord protocol, algorithms, and implementation, which is available at the following URL:
http://www.cs.berkeley.edu/~istoica/papers/2003/chord-ton.pdf

Links to an external site.

Chord uses local information and communicates with other peers to find the host (IP address and port) that a given key is mapped to. Applications may then be built on top of this service that Chord provides. In this project, you will implement a simple distributed storage system for storing text files on top of Chord.

2. Protocol
The Chord protocol and algorithms are described in the paper. You should read the paper to learn about the design of Chord, prior to starting your own implementation.


As discussed in the paper, there are two ways to implement the protocol: iteratively or recursively. Figure 5 in the paper presents the “find successor” function using a recursive implementation. However, it may be easier to approach it iteratively, since then each node will be able to respond to any incoming calls immediately without blocking to wait on responses from other nodes. The following website presents an iterative implementation of the pseudocode for Figure 5 in the paper that you may find helpful (see “Iterative lookups”):
https://cs.utahtech.edu/cs/3410/asst_chord.html

Links to an external site.

3. Chord Client (the basics 10 points)
The Chord client will be a command-line utility which takes the following arguments:
1. -a <String> = The IP address that the Chord client will bind to, as well as advertise to other nodes. Represented as an ASCII string (e.g., 128.8.126.63). Must be specified.
2. -p <Number> = The port that the Chord client will bind to and listen on. Represented as a base-10 integer. Must be specified.
3. --ja <String> = The IP address of the machine running a Chord node. The Chord client will join this node’s ring. Represented as an ASCII string (e.g., 128.8.126.63). Must be specified if --jp is specified.
4. --jp <Number> = The port that an existing Chord node is bound to and listening on. The Chord client will join this node’s ring. Represented as a base-10 integer. Must be specified if --ja is specified.
5. --ts <Number> = The time in milliseconds between invocations of ‘stabilize’. Represented as a base-10 integer. Must be specified, with a value in the range of [1,60000].
6. --tff <Number> = The time in milliseconds between invocations of ‘fix fingers’. Represented as a base-10 integer. Must be specified, with a value in the range of [1,60000].
7. --tcp <Number> = The time in milliseconds between invocations of ‘check predecessor’.
Represented as a base-10 integer. Must be specified, with a value in the range of [1,60000].
8. -r <Number> = The number of successors maintained by the Chord client. Represented as a base-10 integer. Must be specified, with a value in the range of [1,32].
9. -i <String> = The identifier (ID) assigned to the Chord client which will override the ID computed by the SHA1 sum of the client’s IP address and port number. Represented as a string of 40 characters matching [0-9a-fA-F]. Optional parameter.

An example usage to start a new Chord ring is:
chord -a 128.8.126.63 -p 4170 --ts 3000 --tff 1000 --tcp 3000 -r 4


An example usage to join an existing Chord ring is:
chord -a 128.8.126.63 -p 4171 --ja 128.8.126.63 --jp 4170 --ts 3000 --tff 1000 --tcp 3000 -r 4


The Chord client will open a TCP socket and listen for incoming connections on port specified by -p. If neither --ja nor --jp is specified, then the Chord client starts a new ring by invoking ‘create’. The Chord client will initialize the successor list and finger table appropriately (i.e., all will point to the client itself).
Otherwise, the Chord client joins an existing ring by connecting to the Chord client specified by --ja and --jp and invoking ‘join’. The initial steps the Chord client takes when joining the network are described in detail in Section IV.E.1 “Node Joins and Stabilization” of the Chord paper.
Periodically, the Chord client will invoke various stabilization routines in order to handle nodes joining and leaving the network. The Chord client will invoke ‘stabilize’, ‘fix fingers’, and ‘check predecessor’ every --ts, --tff, and --tcp milliseconds, respectively.


Commands:

The Chord client will handle commands by reading from stdin and writing to stdout. There are three command that the Chord client must support: ‘Lookup’, 'StoreFile', and ‘PrintState’.

    ‘Lookup’ takes as input the name of a file to be searched (e.g., “Hello.txt”). The Chord client takes this string, hashes it to a key in the identifier space, and performs a search for the node that is the successor to the key (i.e., the owner of the key). The Chord client then outputs that node’s identifier, IP address, port, and the contents of the file.
    'StoreFile' takes the location of a file on a local disk, then performs a lookup to find the Chord node to store the file at, then uploading the file to the Chord ring.
    ‘PrintState’ requires no input. The Chord client outputs its local state information at the current time, which consists of:
        The Chord client’s own node information and its stored files,
        The node information for all nodes in the successor list,
        The node information for all nodes in the finger table,
        where “node information” corresponds to the identifier, IP address, and port for a given node.

4. Advanced Part: Securing the system (7 points)

The above procedure guarantees no secure way of storing data, as the files can be readable by anyone in the chord network. In this step, we would like to add security in the form of encrypting the files before they get uploaded, using a secure network transfer protocol for messages between nodes, and adding fault-tolerance in the system such that if the host peer dies, the file remains in the network.


5. Cloud Extra (3 points)

Move to the cloud for a few extra points. Here, you should be able to connect to the ring from different machines for full points. You should use a technique such as Docker for simple deployment.

 
Grading

When presenting the lab, you must be able to show at least 6 (preferably 8) clients on the screen at once. We recommend a tool where you can split your terminal, such as Tmux.

Lab responsible: Kåre von Geijer <karev@chalmers.se>