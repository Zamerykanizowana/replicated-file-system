We briefly talked about the algorithm for reconciling transactional conflicts (i.e. node A and node B wrote to the same file at the same time)
and we settled on the **remote always wins** approach.

However, after reading a [paper on DRBD v8](http://citeseerx.ist.psu.edu/viewdoc/download?doi=10.1.1.617.1363&rep=rep1&type=pdf) we started talking about
considering the *Shared disk emulation* algorithm described by the author (which apparently is something that 
[DRBD](https://en.wikipedia.org/wiki/Distributed_Replicated_Block_Device) uses).

Just to recall what the papers says about it:

> 1. A write request is issued on the node, which is sent to the peer, as well as submitted to the local IO
> subsystem (WR).
> 2. The data packet arrives at the peer node and is submitted to the peers io subsystem.
> 3. The write to the peers disk finishes, and an ACK packet is sent back to the origin node.
> 4. The ACK packet arrives at the origin node.
> 5. The local write finishes (local completion).

At the same time, we were in the process of choosing the right communication pattern for [nanomsg](https://nanomsg.org/gettingstarted/index.html) as well.
One of the proposed solutions was to use the [Request/Reply](https://nanomsg.org/gettingstarted/reqrep.html) pattern.
However, this pattern comes at cost of a considerable disadvantage: the need to have separate read and write channels for each node.
Essentially, we would have to emulate a bidirectional synchronous streaming pattern.

While reviewing the patterns, we came across the [Survey](https://nanomsg.org/gettingstarted/survey.html) and while we initially dismissed it
as being useless in the context of our project, it started to make sense the longer we thought about it.

Quickly we realized that any transaction can be considered a voting of some sort.
Let's consider the write operation.

1. Node A wants to write to a file.
1. Node A asks all nodes if they have any objections (e.g. their modification date is newer, the file is currently open by a process, etc).
1. If all nodes oblige, node A can commit the changes.
1. All the other nodes need to know that everybody has agreed as well, so that they can commit the changes made by node A too.

In essence, all nodes are equal when participating in a voting session.
Additionally, all of them need to be aware of the result of the voting.
Thus, we are dealing with an [open ballot system](https://en.wikipedia.org/wiki/Open_ballot_system), i.e. votes are **not** confidential.

This realization, however, disqualifies the aforementioned survey pattern 
because it happens to implement the [secret ballot system](https://en.wikipedia.org/wiki/Secret_ballot), i.e. votes are confidential.

We suspect that the [Bus](https://nanomsg.org/gettingstarted/bus.html) pattern might be the right candidate.
The only thing that we need to make sure of, is that each message contains the information about the transaction ID to which it pertains.
This way we'll be able to know if we have gathered all *ACKS*/*NACKS*.
