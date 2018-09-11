package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"time"
	"sync"
	"labrpc"
)

// import "bytes"
// import "encoding/gob"

const (
	STATE_LEADER = 0
	STATE_CANDIDATE = 1
	STATE_FLLOWER = 2
)


//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make().
//
type ApplyMsg struct {
	Index       int
	Command     interface{}
	UseSnapshot bool   // ignore for lab2; only used in lab3
	Snapshot    []byte // ignore for lab2; only used in lab3
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex
	peers     []*labrpc.ClientEnd
	persister *Persister
	me        int // index into peers[]

	// Your data here.
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	// 查看论文的图2部分，可知

	termTimer		time.Timer 	// 任期计时器
	state 			int			// 当前状态
	voteAcquired 	int 		// 获得的选票

	/*
	 * 全部服务器上面的可持久化状态:
	 *  currentTerm 	服务器看到的最近Term(第一次启动的时候为0,后面单调递增)
	 *  votedFor     	当前Term收到的投票候选 (如果没有就为null)
	 *  log[]        	日志项; 每个日志项包含机器状态和被leader接收的Term(first index is 1)
	*/
	currentTerm int
	votedFor	int
	// log 		bytes[]

	/*
	 * 全部服务器上面的不稳定状态:
	 *	commitIndex 	已经被提交的最新的日志索引(第一次为0,后面单调递增)
	 *	lastApplied     已经应用到服务器状态的最新的日志索引(第一次为0,后面单调递增)
	*/
	// commitIndex int
	// lastApplied int	

	/*
	 * leader上面使用的不稳定状态（完成选举之后需要重新初始化）
	 *	nextIndex[]		对于每一个服务器，需要发送给他的下一个日志条目的索引值	
	 *  matchIndex[]	对于每一个服务器，已经复制给他的日志的最高索引值
	 *
	*/
	// nextIndex 	int
	// matchIndex	int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	var term int
	var isleader bool
	// Your code here.
	term = currentTerm

	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here.
	// Example:
	// w := new(bytes.Buffer)
	// e := gob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	// Your code here.
	// Example:
	// r := bytes.NewBuffer(data)
	// d := gob.NewDecoder(r)
	// d.Decode(&rf.xxx)
	// d.Decode(&rf.yyy)
}




//
// example RequestVote RPC arguments structure.
//
type RequestVoteArgs struct {
	// Your data here.
	term int 			// 候选人的任期号
	candidateId	int		// 请求选票的候选人的Id
	lastLogIndex int	// 候选人的最后日志条目的索引值
	lastLogTerm int		// 候选人最后日志条目的任期号 	
}

//
// example RequestVote RPC reply structure.
//
type RequestVoteReply struct {
	// Your data here.
	term int  			// 当前任期号，以便候选人去更新自己的任期号
	voteGranted bool 	// 候选人赢得了此选票时为真
}

type AppendEntiesArgs struct {
	term int 			// 候选人的任期号
	leaderId	int		// 请求选票的候选人的Id
	preLogIndex int		// 候选人的最后日志条目的索引值
	preLogTerm 	int		// 候选人最后日志条目的任期号 	
	entries 	bytes[] // 准备存储的日志条目（表示心跳时为空）
	leaderCommit 		// 领导人已经提交的日志的索引值
}

type AppendEntiesReply struct {
	term int  			// 当前任期号，用于领导人去更新自己的任期号
	success bool 		// 跟随者包含了匹配prevLogIndex和preLogTerm的日志时为真
}


//
// example RequestVote RPC handler.
//
// 收到投票请求时的处理函数
func (rf *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here.
	if args.term < rf.currentTerm {
		reply.term = rf.currentTerm
		reply.voteGranted = false
	} else if args.term > rf.currentTerm {
		rf.currentTerm = args.term
		rf.state = STATE_CANDIDATE
		rf.votedFor = args.candidateId
		reply.term = rf.currentTerm
		reply.voteGranted = true
	} else {
		if rf.votedFor == -1 {
			rf.votedFor = args.candidateId
			reply.voteGranted = true
		} else {
			reply.voteGranted = flase
		}
	}

	if reply.voteGranted == true {
		go func() {
			rf.vote
		}
	}

}


func (rf *Raft) AppendEntiesArgs(args *AppendEntiesArgs, reply *AppendEntiesReply) {
	if args.term < rf.currentTerm {
		reply.success = false
		reply.term = rf.currentTerm
	} else if args.term > rf.currentTerm {
		rf.currentTerm = args.term
		rf.state = STATE_FLLOWER
		reply.success = true
	} else {
		reply.success = true
	}
	
	
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// returns true if labrpc says the RPC was delivered.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.AppenEntries", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true


	return index, term, isLeader
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
// 建一个Raft端点。
// peers参数是通往其他Raft端点处于连接状态下的RPC连接。
// me参数是自己在端点数组中的索引。
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here.

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())


	return rf
}

// 论文重点：
// 要开始一次选举过程，跟随者先要增加自己的当前任期号并且转换到候选人状态。
// 然后他会并行的向集群中的其他服务器节点发送请求投票的 RPCs 来给自己投票。
// 候选人会继续保持着当前状态直到以下三件事情之一发生：
// (a) 他自己赢得了这次的选举，
// (b) 其他的服务器成为领导者，
// (c) 一段时间之后没有任何一个获胜的人。

// 获得一个随机的任期时间周期
func randTermDuration() {
	r := rand.New(rand.NewSourse(time.Now().UnixNano()))
	return time.Millisecond * time.Duration(r.Int63n(500) + 400)
}

// 开始选举
func (rf *Raft) startElection() {
	rf.currentTerm += 1
	rf.votedFor = rf.me 	// 自己投给自己一票
	rf.voteAcquired = 1		// 自己获得一张选票
	rf.termTimer.Reset(randTermDuration())
	rf.broadcastRequestVote()
}

// 下面这个函数实现的是候选人并行的向其他服务器节点发送请求投票的RPC（sendRequestVote）
func (rf *Raft) broadcastRequestVote() {
	args := RequestVoteArgs{
		Term: rf.currentTerm,
		candidateId: rf.me
	}

	for i, _ := range rf.peers {
		if i == rf.me {
			continue
		}

		go func(server int) {
			var reply RequestVoteReply
			// 如果是候选人就发送投票请求
			if rf.state == STATE_CANDIDATE && rf.sendRequestVote(server, &args, &reply) {
				
				if reply.voteGranted == true {
					rf.voteAcquired += 1
				} else {
					if reply.term > rf.currentTerm {
						rf.currentTerm = reply.Term
						rf.state = STATE_FLLOWER
					}
				}
			}
		}(i)
	}
}

// 下面这个函数实现的是领导者并行的向其他服务器节点发送请求日志（包括心跳）的RPC（sendAppendEntries）
func (rf *Raft) broadcastRequestVote() {
	args := AppendEntiesArgs{
		Term: rf.currentTerm,
		leaderId: rf.me
	}

	for i, _ := range rf.peers {
		if i == rf.me {
			continue
		}

		go func(server int) {
			var reply AppendEntiesReply
			// 如果是候选人就发送投票请求
			if rf.state == STATE_LEADER && rf.sendAppendEntries(server, &args, &reply) {
				if reply.success == true {

				} else {
					if reply.term > rf.currentTerm {
						rf.currentTerm = reply.Term
						rf.state = STATE_FLLOWER
					}
				}
			}
		}(i)
	}
}