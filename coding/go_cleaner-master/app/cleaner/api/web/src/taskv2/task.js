import React, {useEffect, useState} from "react";
import axios from "axios";
import Grid from '@mui/material/Unstable_Grid2';
import {DataGrid} from '@mui/x-data-grid';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import {Button, FormControlLabel, Link, Switch, TextField, Tooltip} from "@mui/material";
import {domain} from "../util/domain";
import {login} from "../util/login";


function TaskPageV2() {
    const statusMap = {"-2": "submitted", "-1": "submitted", 1: "running", 2: "success", 3: "fail", 4: "success"}

    let [task, setTask] = useState([{
        id: -1,
        repo_name: "test/test",
        extra: "{}",
        task_status: -1,
        task_result_mr_link: "https://code.byted.org/xiaoxing.sn/demod_consumer/merge_requests/2"
    }, {id: -2, repo_name: "test/test", extra: "{}", task_status: 2}]);

    const [repo, setRepo] = useState('');
    const inputRepository = (event) => {
        setRepo(event.target.value);
        getDependent(event.target.value);
    }
    const [subDir, setSubDir] = useState('');
    const inputSubDir = (event) => {
        setSubDir(event.target.value);
    }
    const [branch, setBranch] = useState('master');
    const inputBranch = (event) => {
        setBranch(event.target.value);
    }
    const [commitBranch, setCommitBranch] = useState('chore/unused_code_clean');
    const inputCommitBranch = (event) => {
        setCommitBranch(event.target.value);
    }
    const [cleanerFlag, setCleanerFlag] = useState('-f --before=1');
    const inputCleanerFlag = (event) => {
        setCleanerFlag(event.target.value);
    }
    const [endpoint, setEndpoint] = useState('');
    const inputEndpoint = (event) => {
        setEndpoint(event.target.value);
    }
    const [downstreamRepo, setDownstreamRepo] = useState('');
    const inputDownstreamRepo = (event) => {
        setDownstreamRepo(event.target.value);
    }

    let multiGet = function () {
        axios.get(domain() + '/task').then(resp => {
            if (resp.data) {
                setTask(resp.data)
            }
        })
    }
    let submit = function () {
        if (repo === '') {
            alert('blank repository')
            return
        }
        if (repo.includes('*')) {
            alert('invalid repository')
            return
        }
        if (!commitBranch.startsWith('chore/')) {
            alert('commit branch must start with chore/')
            return
        }
        alert('submitted successfully!')
        axios.post(domain() + '/task/submit', {
            'repo_name': repo,
            'extra': JSON.stringify({
                'branch': branch,
                'commit_branch': commitBranch,
                'sub_dir': subDir,
                'cleaner_flag': cleanerFlag,
                'endpoint': endpoint.split(',').filter(token => token.length > 0),
                'downstream_repo':downstreamRepo.split(',').filter(token => token.length > 0),
            })
        }).then(resp => setTask(resp.data))
    }
    let del = function (task) {
        axios.post(domain() + '/task/delete', task).then(resp => setTask(resp.data))
    }

    let getDependent = function (repo) {
        axios.get(domain() + '/task/repo/dependent?repo='+repo).then(resp => setDownstreamRepo(resp.data.join(',')))
    }

    let [allowDel, setAllowDel] = useState(false)


    useEffect(() => {
        login()
        multiGet()
    }, [])

    const columns = [
        {field: 'id', headerName: 'ID', width: 80},
        {field: 'repo_name', headerName: 'Repository', width: 150},
        {
            field: 'extra', headerName: 'Detail', width: 150, renderCell: (params) => (
                <Tooltip title={params.value} placement="top">
                    <span className="csutable-cell-trucate">{params.value}</span>
                </Tooltip>
            ),
        },
        {
            field: 'task_status', headerName: 'Status', width: 100, renderCell: (params) => (
                <span>{statusMap[params.value]}</span>
            )
        },
        {field: 'task_result_unused_line', headerName: 'Del-LoC', width: 100},
        {field: 'task_result_total_line', headerName: 'LoC', width: 100},
        {field: 'task_result_err_msg', headerName: 'ErrMsg', width: 100},
        {field: 'task_result_stderr', headerName: 'StdErr', width: 100},
        {
            field: 'task_result_mr_link',
            headerName: 'MR',
            width: 130,
            renderCell: (params) => (
                <Link href={params.value} target="_blank" rel="noopener noreferrer">
                    {params.value != null && params.value.length > 0 ? "Link" : ""}
                </Link>
            ),
        },
        {field: 'created_at', headerName: 'Create', width: 150},
        {field: 'updated_at', headerName: 'Update', width: 150},
        {
            field: "delete",
            headerName: "Delete",
            sortable: false,
            width: 100,
            disableClickEventBubbling: true,
            renderCell: (params) => {
                const onClick = () => {
                    if (params.row.task_status === -1 && allowDel) {
                        del(params.row)
                    }
                };

                return <Button variant="contained" onClick={onClick}>Del</Button>;
            }
        }
    ];


    return (
        <div>
            <Grid container spacing={2}>
                <Grid xs={8}>
                    <Toolbar>
                        <Typography variant="h6" component="div" sx={{flexGrow: 1}}>
                            Task Queue
                        </Typography>
                        <FormControlLabel
                            control={<Switch defaultChecked checked={allowDel} onChange={(event, checked) => {
                                setAllowDel(checked)
                            }}/>} label="ALLOW DEL"/>
                    </Toolbar>
                    <DataGrid
                        rows={task}
                        columns={columns}
                        initialState={{
                            pagination: {
                                paginationModel: {page: 0, pageSize: 100},
                            },
                        }}
                        // checkboxSelection
                        rowSelection={true}

                    />
                </Grid>
                <Grid xs={4}>
                    <Toolbar>
                        <Typography variant="h6" component="div" sx={{flexGrow: 0}}>
                            Submit Task
                        </Typography>
                    </Toolbar>
                    <Toolbar>
                        <TextField label="repository name, eg: tiktok/item_data" variant="outlined" multiline
                                   rows={1} fullWidth onChange={inputRepository} value={repo}/>
                    </Toolbar>
                    <Toolbar>
                        <TextField label="subdirectory, eg: app/mention_biz. Only for monorepo." variant="outlined"
                                   rows={1} fullWidth onChange={inputSubDir} value={subDir}/>
                    </Toolbar>
                    <Toolbar>
                        <TextField label="branch // 在该分支基础上清理" variant="outlined"
                                   rows={1} fullWidth onChange={inputBranch} value={branch}/>
                    </Toolbar>
                    <Toolbar>
                        <TextField
                            label="commit branch, must start with 'chore/' // 清理后提交到该分支, 分支名必须以'chore/'开头"
                            variant="outlined"
                            rows={1} fullWidth onChange={inputCommitBranch} value={commitBranch}/>
                    </Toolbar>
                    <Toolbar>
                        <TextField
                            label="unused method/URL, separated by commas // 无用的接口名或URL, 逗号分割"
                            multiline={true}
                            style={{paddingBottom: '10px', paddingTop: '10px'}}
                            rows={4} fullWidth onChange={inputEndpoint} value={endpoint}/>
                    </Toolbar>
                    <Toolbar>
                        <TextField
                            label="repositories import this, eg: a/a,b/b // 依赖这个仓库的其它仓库, 逗号分割"
                            multiline={true}
                            style={{paddingBottom: '10px', paddingTop: '10px'}}
                            rows={4} fullWidth onChange={inputDownstreamRepo} value={downstreamRepo}/>
                    </Toolbar>
                    <Toolbar>
                        <TextField
                            label="cleaner tool flag // cleaner工具的参数，详见https://zjsms.com/iYUyKNfH/"
                            variant="outlined"
                            rows={1} fullWidth onChange={inputCleanerFlag} value={cleanerFlag}/>
                    </Toolbar>
                    <Toolbar>
                        <Button variant="contained" fullWidth onClick={submit}>Submit</Button>
                    </Toolbar>
                </Grid>
            </Grid>
        </div>
    );
}

export default TaskPageV2;
