import { Routes, Route, Navigate } from 'react-router-dom'
import Dashboard from '../pages/student/Dashboard'
import Exam from '../pages/student/Exam'
import Practice from '../pages/student/Practice'
import Task from '../pages/student/Task'
import Report from '../pages/student/Report'
import Project from '../pages/student/Project'
import ProjectStage from '../pages/student/ProjectStage'
import Defense from '../pages/student/Defense'
import Grades from '../pages/student/Grades'
import Appeal from '../pages/student/Appeal'

export default function StudentRouter() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/student/dashboard" replace />} />
      <Route path="/student/dashboard" element={<Dashboard />} />
      <Route path="/student/exam/:id" element={<Exam />} />
      <Route path="/student/practice/:id" element={<Practice />} />
      <Route path="/student/task/:id" element={<Task />} />
      <Route path="/student/report/:id" element={<Report />} />
      <Route path="/student/project/:id" element={<Project />} />
      <Route path="/student/project/:id/stage/:stageId" element={<ProjectStage />} />
      <Route path="/student/defense/:id" element={<Defense />} />
      <Route path="/student/grades" element={<Grades />} />
      <Route path="/student/appeal/:id" element={<Appeal />} />
      <Route path="*" element={<Navigate to="/student/dashboard" replace />} />
    </Routes>
  )
}
