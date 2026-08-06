import React, {useEffect, useState} from "react";
import axios from "axios";
import Grid from '@mui/material/Unstable_Grid2';
import {DataGrid} from '@mui/x-data-grid';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import {Button, FormControlLabel, Switch, TextField, Tooltip} from "@mui/material";
import {domain} from "../util/domain";
import {login} from "../util/login";


function TaskPage() {
    let [task, setTask] = useState([]);

    const [repo, setRepo] = useState('');
    const inputRepository = (event) => {
        setRepo(event.target.value);
    }
    const [opt, setOpt] = useState('{}');
    const inputOpt = (event) => {
        setOpt(event.target.value);
    }

    let multiGet = function () {
        axios.get(domain() + '/task').then(resp => setTask(resp.data))
    }
    let submit = function () {
        try {
            JSON.parse(opt);
        } catch (error) {
            alert("Invalid JSON");
            return;
        }
        axios.post(domain() + '/task/submit', {'repo_name': repo, 'task_opt': opt}).then(resp => setTask(resp.data))
    }
    let del = function (task) {
        axios.post(domain() + '/task/delete', task).then(resp => setTask(resp.data))
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
            field: 'task_opt', headerName: 'Opt', width: 70, renderCell: (params) => (
                <Tooltip title={params.value} placement="top">
                    <span className="csutable-cell-trucate">{params.value}</span>
                </Tooltip>
            ),
        },
        {field: 'task_status', headerName: 'Status', width: 70},
        {field: 'task_result_unused_line', headerName: 'Unused LoC', type: 'number', width: 120},
        {field: 'task_result_total_line', headerName: 'LoC', type: 'number', width: 120},
        {field: 'task_result_err_msg', headerName: 'Err', width: 80},
        {field: 'task_result_mr_link', headerName: 'MR', width: 150},
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
                        <TextField label="repository, multiple separated by commas" variant="outlined" multiline
                                   rows={10} fullWidth onChange={inputRepository} value={repo}/>
                    </Toolbar>
                    <Toolbar style={{marginTop: '10px'}}>
                        <TextField label="task opt, json format" variant="outlined" multiline
                                   rows={10} fullWidth onChange={inputOpt} value={opt}/>
                    </Toolbar>
                    <Toolbar>
                        <Button variant="contained" fullWidth onClick={submit}>Submit</Button>
                    </Toolbar>
                </Grid>
            </Grid>
        </div>
    );
}

export default TaskPage;
