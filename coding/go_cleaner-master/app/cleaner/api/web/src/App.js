import React from "react";
import {BrowserRouter as Router, Route, Routes} from "react-router-dom";
import TaskPageV2 from "./taskv2/task";
import ABEntryPage from "./ab/entry/entry";

function App() {
    return (
        <Router>
            <Routes>
                <Route path="/" element={<TaskPageV2/>}/>
                {/*<Route path="/task" element={<TaskPageV2 />} />*/}
                <Route path="/abentry" element={<ABEntryPage/>}/>
            </Routes>
        </Router>
    );
}

export default App;