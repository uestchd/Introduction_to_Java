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

import "sync"
import "labrpc"
import "time"
import "math/rand"
// import "bytes"
// import "encoding/gob"



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
const (
	HeartbeatTime  = 100
	ElectionMinTime = 150
	ElectionMaxTime = 300
)
//
type LogEntry struct {
    Command interface{}
	Term    int
}
//
type AppendEntryReply struct {
	Term        int
	Success     bool
	CommitIndex int
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

	// Persistent state on all servers
	currentTerm int 
	votedFor    int 
	logs        []LogEntry 

	// Volatile state on all servers
	commitIndex int
	lastApplied int 

	// Volatile state on leaders
	nextIndex  []int 
	matchIndex []int 

	// votes COUNT
	votesCount int

	state   string
	applyCh chan ApplyMsg

	timer *time.Timer

}
//
type AppendEntryArgs struct {
	Term         int
	Leader_id    int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}
//
func (rf *Raft) restartTime() {

	randst := ElectionMinTime+rand.Int63n(ElectionMaxTime-ElectionMinTime)
	timeout := time.Millisecond * time.Duration(randst)
	if rf.state == "LEADER" {
		timeout = HeartbeatTime * time.Millisecond
		randst = HeartbeatTime
	}
	if rf.timer == nil {
		rf.timer = time.NewTimer(timeout)
		go func() {
			for {
				<-rf.timer.C

				rf.Timeout()
			}
		}()
	}
	rf.timer.Reset(timeout)
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here.
	term = rf.currentTerm
	isLeader = (rf.state == "LEADER")

	return term, isLeader
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
	Term         int 
	CandidateId  int 
	LastLogIndex int 
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
//
type RequestVoteReply struct {
	// Your data here.
	Term        int  
	VoteGranted bool 
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here.
	rf.mu.Lock()
	defer rf.mu.Unlock()


	can_vote := true


	if len(rf.logs)>0{

		if rf.logs[len(rf.logs)-1].Term > args.LastLogTerm{
			can_vote = false

		}
		if rf.logs[len(rf.logs)-1].Term == args.LastLogTerm && len(rf.logs)-1 > args.LastLogIndex {
			can_vote = false
		}
	}

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		//fmt.Printf("\nreceive request: "+strconv.Itoa(rf.me)+" "+ strconv.Itoa(len(rf.logs))+" "+ strconv.Itoa(rf.currentTerm))
		return
	}

	if args.Term == rf.currentTerm {
		if rf.votedFor == -1 && can_vote{
			rf.votedFor = args.CandidateId
		}
		reply.Term = rf.currentTerm
		reply.VoteGranted = (rf.votedFor == args.CandidateId)
		//fmt.Printf("\nreceive request: "+strconv.Itoa(rf.me)+" "+ strconv.Itoa(len(rf.logs))+" "+ strconv.Itoa(rf.currentTerm))

		return
	}

	if args.Term > rf.currentTerm {

		rf.state = "FOLLOWER"
		rf.currentTerm = args.Term
		rf.votedFor = -1

		if(can_vote){
		rf.votedFor = args.CandidateId
		}

		rf.restartTime()
		//fmt.Printf("\nrestart time in RequestVote: "+strconv.Itoa(rf.me))

		reply.Term = args.Term
		reply.VoteGranted = (rf.votedFor == args.CandidateId)
		//fmt.Printf("\nreceive request: "+strconv.Itoa(rf.me)+" "+ strconv.Itoa(len(rf.logs))+" "+ strconv.Itoa(rf.currentTerm))

		return
	}
}

//
// handle vote result
//
func (rf *Raft) countVote(reply RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if reply.Term < rf.currentTerm {
		return
	}


	if reply.Term > rf.currentTerm {
		rf.currentTerm = reply.Term
		rf.state = "FOLLOWER"
		rf.votedFor = -1
		rf.restartTime()
		//fmt.Printf("\nrestart time in countVotessss: "+strconv.Itoa(rf.me))
		return
	}


	if rf.state == "CANDIDATE" && reply.VoteGranted {
		rf.votesCount += 1
		if rf.votesCount >= (len(rf.peers))/2 + 1 {
			//fmt.Printf("\nLEADER: "+strconv.Itoa(rf.me)+" Term: "+strconv.Itoa(rf.currentTerm))
			rf.state = "LEADER"
			for i := 0; i < len(rf.peers); i++ {
				if i == rf.me {
					continue
				}
				rf.nextIndex[i] = len(rf.logs)
				rf.matchIndex[i] = -1
			}
			//fmt.Printf("\nrestart time in countVote: "+strconv.Itoa(len(rf.logs)))
			//rf.SendAppendEntries()
			rf.restartTime()
			//fmt.Printf("\nrestart time in countVote: "+strconv.Itoa(rf.me))
		}
		return
	}
}
//
// append entries
//
func (rf *Raft) AppendEntries(args AppendEntryArgs, reply *AppendEntryReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	//fmt.Printf("\nAppendEntries: "+strconv.Itoa(rf.me)+" Term: "+strconv.Itoa(rf.currentTerm))
	if args.Term < rf.currentTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
		//fmt.Printf("\nAppendEntries:args.Term < rf.currentTerm ")
	} else {

		rf.state = "FOLLOWER"
		rf.currentTerm = args.Term
		rf.votedFor = -1
		reply.Term = args.Term

		if args.PrevLogIndex >=0 && (len(rf.logs)-1 <args.PrevLogIndex || rf.logs[args.PrevLogIndex].Term != args.PrevLogTerm){
			index := len(rf.logs)-1
			if index>args.PrevLogIndex{
				index = args.PrevLogIndex
			}

			for index >=0 {
				if(args.PrevLogTerm == rf.logs[index].Term){
				 break
			  }
				index --
			}
			reply.CommitIndex = index
			reply.Success = false
		}else if args.Entries != nil {

			rf.logs = rf.logs[:args.PrevLogIndex+1]
			rf.logs = append(rf.logs, args.Entries...)
			if len(rf.logs)-1 >= args.LeaderCommit {
				rf.commitIndex = args.LeaderCommit
				go rf.commit()
			}
			reply.CommitIndex = len(rf.logs) - 1
			reply.Success = true
		} else {
			if len(rf.logs)-1 >= args.LeaderCommit {
				rf.commitIndex = args.LeaderCommit
				go rf.commit()
			}
			reply.CommitIndex = args.PrevLogIndex
			reply.Success = true
		}

	}
	rf.restartTime()
	//fmt.Printf("\nrestart timer in AppendEntry: "+strconv.Itoa(rf.me))
}
//
func (rf *Raft) commit() {

	rf.mu.Lock()
	defer rf.mu.Unlock()

	i := rf.lastApplied + 1
	for i <= rf.commitIndex{

		var args ApplyMsg
		args.Index = i+1
		args.Command = rf.logs[i].Command
		rf.applyCh <- args
		i++
	}
	rf.lastApplied = rf.commitIndex

}
//
func (rf *Raft) SendAppendEntries() {

	//fmt.Printf("\nSendAppendEntries: "+strconv.Itoa(rf.me))
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}
		var args AppendEntryArgs
		args.Term = rf.currentTerm
		args.Leader_id = rf.me
		//fmt.Printf("\nSendAppendEntries:1234 "+strconv.Itoa(rf.me))
		args.PrevLogIndex = rf.nextIndex[i]-1
		//fmt.Printf("\nSendAppendEntries:1234 "+strconv.Itoa(rf.me))
		if args.PrevLogIndex >=0 {
			//fmt.Printf("\nSendAppendEntries:12345 "+strconv.Itoa(args.PrevLogIndex)+" LEN: "+strconv.Itoa(len(rf.logs)))
			args.PrevLogTerm = rf.logs[args.PrevLogIndex].Term
			//fmt.Printf("\nSendAppendEntries:12346 "+strconv.Itoa(rf.me))
		}
		//fmt.Printf("\nSendAppendEntries:1234 "+strconv.Itoa(rf.me))
		if rf.nextIndex[i] < len(rf.logs) {
			args.Entries = rf.logs[rf.nextIndex[i]:]
		}

		args.LeaderCommit = rf.commitIndex
		//fmt.Printf("\nSendAppendEntries:123 "+strconv.Itoa(rf.me))
		go func(server int, args AppendEntryArgs) {
			var reply AppendEntryReply
			ok := rf.peers[server].Call("Raft.AppendEntries", args, &reply)
			if ok {
				rf.handleAppendEntries(server, reply)
			}
		}(i, args)
	}
}
//
//
// Handle AppendEntry result
//
func (rf *Raft) handleAppendEntries(server int, reply AppendEntryReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	//fmt.Printf("\nhandleAppendEntries: "+strconv.Itoa(rf.me))
	if rf.state != "LEADER" {
		//fmt.Printf("\nhandleAppendEntries: is not leader"+strconv.Itoa(rf.me))
		return
	}

	// "LEADER" should degenerate to Follower
	if reply.Term > rf.currentTerm {
		rf.currentTerm = reply.Term
		rf.state = "FOLLOWER"
		rf.votedFor = -1
		rf.restartTime()
		//fmt.Printf("\nrestart time in handleAppendEntries: reply.Term > rf.currentTerm "+strconv.Itoa(rf.me))
		return
	}

	if reply.Success{
		//fmt.Printf("\nhandleAppendEntries: is true "+strconv.Itoa(rf.me))
		rf.nextIndex[server] = reply.CommitIndex + 1
		rf.matchIndex[server] = reply.CommitIndex
		count :=1
		i :=0
		for i < len(rf.peers){
			if i!=rf.me && rf.matchIndex[i] >= rf.matchIndex[server]{
				count += 1
			}
			i++
			//fmt.Printf("\nhandleAppendEntries: "+strconv.Itoa(i))
		}
		if count >= len(rf.peers)/2+1{
			if rf.commitIndex < rf.matchIndex[server] &&
			rf.logs[rf.matchIndex[server]].Term == rf.currentTerm {
				rf.commitIndex = rf.matchIndex[server]
				go rf.commit()
			}
		}

	}else{
		//fmt.Printf("\nhandleAppendEntries: is false"+strconv.Itoa(rf.me))
		rf.nextIndex[server] = reply.CommitIndex + 1
		rf.SendAppendEntries()
	}
}
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
	rf.mu.Lock()
	defer rf.mu.Unlock()

	index := -1
	term := -1
	isLeader := false

	//fmt.Printf("\ntimeout: "+strconv.Itoa(rf.me)+" "+rf.state)
	if rf.state !="LEADER"{
		return index, term, isLeader
	}

	var newlog LogEntry
	newlog.Command = command
	newlog.Term = rf.currentTerm
	rf.logs = append(rf.logs,newlog)
	index = len(rf.logs)
	isLeader = true
	term = rf.currentTerm


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
//
// when peer timeout, it changes to be a candidate and sendRequwstVote.
//
func (rf *Raft) Timeout() {

	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state != "LEADER" {
		rf.state = "CANDIDATE"
		//fmt.Printf("\ncandidate:" +strconv.Itoa(rf.me))
		rf.currentTerm += 1
		rf.votedFor = rf.me
		rf.votesCount = 1

		var args RequestVoteArgs
		args.Term = rf.currentTerm
		args.CandidateId = rf.me
		args.LastLogIndex = len(rf.logs) -1

		if args.LastLogIndex>=0{
			args.LastLogTerm = rf.logs[args.LastLogIndex].Term
		}

		for peer := 0; peer < len(rf.peers); peer++ {
			if peer == rf.me {
				continue
			}

			go func(peer int, args RequestVoteArgs) {
				var reply RequestVoteReply
				ok := rf.peers[peer].Call("Raft.RequestVote", args, &reply)
				if ok {
					rf.countVote(reply)
				}
			}(peer, args)

		}
	} else {
		rf.SendAppendEntries()
		//fmt.Printf("\nSendAppendEntries: "+strconv.Itoa(rf.me)+" "+strconv.Itoa(rf.currentTerm)+" "+rf.state)
	}
	rf.restartTime()
	//fmt.Printf("\nrestart time in Timeout: "+strconv.Itoa(rf.me))
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
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here.
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.state = "FOLLOWER"
	rf.applyCh = applyCh

	rf.logs = make([]LogEntry,0)
	rf.commitIndex = -1
	rf.lastApplied = -1

	rf.nextIndex = make([]int,len(peers))
	rf.matchIndex = make([]int,len(peers))
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	rf.restartTime()

	return rf
}
