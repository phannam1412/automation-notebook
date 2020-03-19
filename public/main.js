class Main extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            history: HISTORY,
            running_process_ids: [],
            viewing_process_ids: [],
            finished_jobs: [],
            manual_scroll: false,
        }
    }

    post(uri, body) {
        return new Promise((resolve, reject) => {
            var xhr = new XMLHttpRequest();
            xhr.open('POST', uri, true);
            xhr.onload = function () {
                if (xhr.status < 400)
                    resolve(xhr.responseText);
                else
                    reject(xhr.responseText);
            };
            xhr.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
            xhr.send(body);
        });
    }

    get(uri, body) {
        return new Promise((resolve, reject) => {
            let xhr = new XMLHttpRequest();
            xhr.open('GET', uri, true);
            xhr.onload = function () {
                if (xhr.status < 400)
                    resolve(xhr.responseText);
                else
                    reject(xhr.responseText);
            };
            xhr.send(body);
        });
    }

    async loadStatus() {
        let res = await this.get('status');
        console.log(res);
        res = JSON.parse(res);
        // are there some processes that are viewing by client but not running anymore on server ?
        // we need to remove all these watches
        // let running_process_ids = res.running_process_ids;
        // let finished_jobs = res.finished_jobs;
        // let valid_viewing_process_ids = [];
        // let viewing_process_ids = this.state.viewing_process_ids;
        // for(let a=0;a<viewing_process_ids.length;a++) {
        //     if(running_process_ids.indexOf(viewing_process_ids[a]) >= 0) {
        //         valid_viewing_process_ids.push(viewing_process_ids[a]);
        //         continue;
        //     }
        // }
        this.setState({running_process_ids: res.running_process_ids, finished_jobs: res.finished_jobs});
    }

    componentDidMount() {
        var that = this;
        setInterval(function(){
            if(that.state.manual_scroll)
                return;
            // scroll the output to bottom
            let output = $('#output').contents();
            output.scrollTop(output.height());
        }, 200);

        $("body").height(window.innerHeight);
        $( "#query" ).autocomplete({
            serviceUrl: "search",
            dataType: "JSON",
            triggerSelectOnValidInput: false,
            onSelect: function (suggestion) {
                console.log(suggestion);
                that.setState({text: suggestion.value});
            }
        });

        // get server status intervally
        setInterval(() => this.loadStatus(), 1000);
    }

    async runCommand(command) {
        await this.post('run', 'command=' + encodeURIComponent(command));
        let history = this.state.history;
        history.unshift(command);
        this.setState({text: '', history});
        this.loadStatus();
    }

    handleKeyDownOnSearchInput(e) {
        console.log(e);
        if(e.keyCode === 13) {
            let command = this.state.text;
            if(e.shiftKey) {
                // runCommand(command);
            } else {
                this.runCommand(command);
            }
        }
    }

    async closeProcess(process_id) {
        await this.post('close-process', 'process_id=' + process_id);
        this.loadStatus();
    }

    render() {
        console.log('rendering Main...');
        return (
            <div className="container" style={{height: '99%', display: 'flex'}}>
                <div style={{flex: 100, height: '100%', float: 'left', display: 'flex', flexDirection: 'column'}}>
                    <div style={{flex: 1}}>
                        <input style={{padding: '1%', width: '98%'}} type="text" id="query" name="query"
                               onChange={e => this.setState({text: e.target.value})}
                               onKeyDown={e => this.handleKeyDownOnSearchInput(e)}
                               placeholder="what do you want ?" value={this.state.text}>
                        </input>
                    </div>
                    <div style={{flex: 1}}>
                        <input type="checkbox" title="manual scroll" onChange={() => this.setState({manual_scroll: !this.state.manual_scroll})} />
                        <span>manual scroll</span>
                    </div>
                    <iframe id="output" src={"/log?process_id=" + this.state.viewing_process_ids.join(',')} style={{width: '100%', flex: 100, backgroundColor:'white', color: 'black'}} />
                </div>
                <div style={{flex: 1}} />
                <div style={{flex: 30, height: '100%', float: 'right', overflowY: 'scroll', backgroundColor: 'black', color: 'white'}}>
                    <JobCollection
                        key={1}
                        processes={this.state.finished_jobs}
                        viewingProcessIds={this.state.viewing_process_ids}
                        onWatchChange={ids => this.setState({viewing_process_ids: ids})}
                        willClose={process_id => this.closeProcess(process_id)}
                        backgroundColor='#444444'
                        disableTimer={true}
                    />
                    <JobCollection
                        key={2}
                        processes={this.state.running_process_ids}
                        viewingProcessIds={this.state.viewing_process_ids}
                        onWatchChange={ids => this.setState({viewing_process_ids: ids})}
                        willClose={process_id => this.closeProcess(process_id)}
                    />
                    <History history={this.state.history}
                             onItemClicked={command => this.runCommand(command)}
                    />
                </div>
            </div>
        )
    }
}

class History extends React.Component {
    constructor(props) {
        super(props);
    }
    render() {
        return (
            <div style={{paddingLeft: '5%', paddingRight: '5%'}} className="command-container">
                {
                    this.props.history.map(
                        (command, key) => <p onClick={e => this.props.onItemClicked(command)}
                                             key={key}
                                             style={{cursor: 'pointer'}}>
                            {command}
                        </p>
                    )
                }
            </div>
        )
    }
}

class JobCollection extends React.Component {
    constructor(props) {
        super(props);
    }
    filterUnique(arr) {
        return arr.filter(function onlyUnique(value, index, self) {
            return self.indexOf(value) === index;
        });
    }
    render() {
        console.log('render JobCollection', this.props.processes, this.props.viewingProcessIds);
        return (
            <div style={{padding: '5%', backgroundColor: this.props.backgroundColor || '#222222'}} className="job-container">
                {
                    this.props.processes && this.props.processes.map(item => {
                        let process_id = item.process_id;
                        let command = item.command;
                        return (
                            <Job key={process_id}
                                 isWatching={this.props.viewingProcessIds.indexOf(process_id) >= 0}
                                 command={command}
                                 processId={process_id}
                                 disableTimer={this.props.disableTimer}
                                 startWatch={() => {
                                     let ids = this.props.viewingProcessIds;
                                     ids.push(process_id);
                                     ids = this.filterUnique(ids);
                                     this.props.onWatchChange(ids);
                                 }}
                                 unWatch={() => {
                                     let ids = this.props.viewingProcessIds;
                                     let index = ids.indexOf(process_id);
                                     if(index < 0)
                                         return;
                                     ids.splice(index, 1);
                                     ids = this.filterUnique(ids);
                                     this.props.onWatchChange(ids);
                                 }}
                                 willClose={() => this.props.willClose(process_id)}
                            />
                        )
                    })
                }
            </div>
        )
    }
}

class Job extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            time: 0,
            timeDisplay: '',
            isClosing: false,
        };
    }
    componentDidMount() {
        if(this.props.disableTimer)
            return;
        setInterval(() => {
            let time = this.state.time + 1;
            let min = Math.floor(time / 60);
            let sec = time % 60;
            if(min === 0)
                this.setState({time, timeDisplay: sec.toString() + "s"});
            else
                this.setState({time, timeDisplay: min.toString() + "m" + sec.toString() + "s"});
        }, 1000);
    }
    click(e) {
        if(this.props.isWatching)
            this.props.unWatch();
        else
            this.props.startWatch();
    }
    render() {
        let ICON_EYE = '👁 ';
        return (
            <p style={{cursor: 'pointer', textDecoration: this.state.isClosing ? 'line-through': ''}} onClick={e => this.click(e)}>
                <span onClick={() => {
                    this.props.willClose();
                    this.setState({isClosing: true})
                }}>(×) </span>
                <span style={{fontSize: 9}}>{this.state.timeDisplay} {this.props.isWatching ? ICON_EYE : ''}</span>
                &nbsp;&nbsp;
                <span>{this.props.command}</span>
            </p>
        )
    }
}