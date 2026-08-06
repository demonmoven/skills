import React, {useEffect, useState} from "react";
import axios from "axios";
import Grid from '@mui/material/Unstable_Grid2';
import {DataGrid} from '@mui/x-data-grid';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import {
    Button,
    Dialog,
    DialogTitle,
    FormControl,
    FormControlLabel,
    InputLabel,
    MenuItem,
    OutlinedInput,
    Select,
    styled,
    Switch,
    TextField
} from "@mui/material";
import {domain} from "../../util/domain";
import {login} from "../../util/login";


function ABEntryPage() {

    let [entries, setEntries] = useState([]); // table
    let [entry, setEntry] = React.useState({}); // save one

    let multiGet = function () {
        axios.get(domain() + '/ab/entry').then(resp => setEntries(resp.data))
    }
    let submit = function () {
        axios.post(domain() + '/ab/entry/save', entry).then(resp => setEntries(resp.data))
    }
    let del = function (task) {
        axios.post(domain() + '/ab/entry/delete', task).then(resp => setEntries(resp.data))
    }

    const Dialog1 = styled(Dialog)({
        '& .MuiDialog-paper': {
            width: '150%',
            height: '80%',
        },
    });


    let [allowDel, setAllowDel] = useState(false)
    const [open, setOpen] = React.useState(false);

    useEffect(() => {
        login()
        multiGet()
    }, [])

    const columns = [
        {field: 'id', headerName: 'ID', width: 40},
        {field: 'package', headerName: 'Go Package', width: 250},
        {field: 'recv', headerName: 'Receiver', width: 100},
        {field: 'func', headerName: 'Function', width: 250},
        {field: 'key_index', headerName: 'KeyPos', width: 100},
        {field: 'default_index', headerName: 'DefaultPos', width: 80},
        {field: 'key_prefix', headerName: 'KeyPrefix', width: 150},
        {field: 'default_val', headerName: 'Default', width: 150},
        {field: 'default_typ', headerName: 'DefaultType', width: 150},
        {field: 'author', headerName: 'Operator', width: 250},
        {
            field: "copy",
            headerName: "Copy",
            sortable: false,
            width: 100,
            disableClickEventBubbling: true,
            renderCell: (params) => {
                const onClick = () => {
                    setEntry({...JSON.parse(JSON.stringify(params.row)), id: 0})
                    setOpen(true)
                };
                return <Button variant="contained" onClick={onClick}>Copy</Button>;
            }
        },
        {
            field: "update",
            headerName: "Update",
            sortable: false,
            width: 100,
            disableClickEventBubbling: true,
            renderCell: (params) => {
                const onClick = () => {
                    setEntry(JSON.parse(JSON.stringify(params.row)))
                    setOpen(true)
                };
                return <Button variant="contained" onClick={onClick}>Update</Button>;
            }
        },
        {
            field: "delete",
            headerName: "Delete",
            sortable: false,
            width: 100,
            disableClickEventBubbling: true,
            renderCell: (params) => {
                const onClick = () => {
                    if (allowDel) {
                        del(params.row)
                    }
                };

                return <Button variant="contained" onClick={onClick}>Del</Button>;
            }
        },
    ];


    return (
        <div>
            <Grid>
                <Toolbar>
                    <Typography variant="h6" component="div" sx={{flexGrow: 1}}>
                        AB Entry Configuration
                    </Typography>
                    <FormControlLabel
                        control={<Switch defaultChecked checked={allowDel} onChange={(event, checked) => {
                            setAllowDel(checked)
                        }}/>} label="ALLOW DEL"/>
                </Toolbar>
                <DataGrid
                    rows={entries}
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
            <Grid>
                <Grid>
                    <Dialog1 onClose={() => {
                        setOpen(false)
                    }} open={open}>
                        <DialogTitle></DialogTitle>
                        <Grid>
                            <Toolbar>
                                <TextField
                                    required
                                    fullWidth
                                    label="GoPackage"
                                    defaultValue={entry.package}
                                    onChange={(value) => {
                                        entry.package = value.target.value
                                    }}
                                />
                            </Toolbar>
                            <Toolbar>
                                <TextField
                                    required
                                    fullWidth
                                    label="Recevier"
                                    defaultValue={entry.recv}
                                    onChange={(value) => {
                                        entry.recv = value.target.value
                                    }}
                                />
                            </Toolbar>
                            <Toolbar>
                                <TextField
                                    required
                                    fullWidth
                                    label="Func (Seperated by comma)"
                                    defaultValue={entry.func}
                                    onChange={(value) => {
                                        entry.func = value.target.value.split(',')
                                    }}
                                />
                            </Toolbar>
                            <Toolbar>
                                <TextField
                                    required
                                    fullWidth
                                    label="KeyPos (Seperated by comma)"
                                    defaultValue={entry.key_index}
                                    onChange={(value) => {
                                        entry.key_index = value.target.value.split(',').map((i) => parseInt(i))
                                    }}
                                />
                            </Toolbar>
                            <Toolbar>
                                <TextField
                                    fullWidth
                                    label="DefaultPos"
                                    defaultValue={entry.default_index}
                                    onChange={(value) => {
                                        entry.default_index = parseInt(value.target.value)
                                    }}
                                />
                            </Toolbar>
                            <Toolbar>
                                <TextField
                                    fullWidth
                                    label="KeyPrefix"
                                    defaultValue={entry.key_prefix}
                                    onChange={(value) => {
                                        entry.key_prefix = value.target.value
                                    }}
                                />
                            </Toolbar>
                            <Toolbar>
                                <TextField
                                    fullWidth
                                    label="Default"
                                    defaultValue={entry.default_val}
                                    onChange={(value) => {
                                        entry.default_val = value.target.value
                                    }}
                                />
                            </Toolbar>
                            <Toolbar>
                                <FormControl fullWidth>
                                    <InputLabel id="DefaultType">DefaultType</InputLabel>
                                    <Select
                                        labelId="DefaultType"
                                        id="DefaultType"
                                        label="DefaultType"
                                        onChange={(value) => {
                                            entry.default_typ = value.target.value
                                        }}
                                        input={<OutlinedInput label="Name"/>}
                                        defaultValue={entry.default_typ}
                                        fullWidth
                                    >
                                        {['int', 'float', 'string', 'bool'].map((name) => (
                                            <MenuItem
                                                key={name}
                                                value={name}
                                            >
                                                {name}
                                            </MenuItem>
                                        ))}
                                    </Select>
                                </FormControl>
                            </Toolbar>

                            <Toolbar>
                                <Button variant="contained" fullWidth onClick={() => {
                                    submit()
                                    setOpen(false)
                                }}>Submit</Button>
                            </Toolbar>

                        </Grid>
                    </Dialog1>
                </Grid>
            </Grid>

        </div>
    );
}

export default ABEntryPage;
