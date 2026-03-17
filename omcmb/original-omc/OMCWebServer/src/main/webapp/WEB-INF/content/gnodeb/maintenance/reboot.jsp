<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<!DOCTYPE html>
<html>

<head>
    <title>Reboot</title>
    <style>
        #gnodeb_reboot_ctn .el-date-editor .el-range__close-icon {
            line-height: 20px;
            font-size: 12px;
        }

        #gnodeb_reboot_ctn .el-date-editor .el-range-input {
            font-size: 12px;
        }
    </style>
</head>

<body>
    <!-- gNB重启 -->
    <div id="gnodeb_reboot_ctn" class="container commonWarp">
        <div class="el-icon-copy-document"></div>
        <el-tabs class="fit newTabs">
            <el-tab-pane label="<%=rb.getString("ChongQi")%>">
                <!-- 操作按钮 -->
                <div class="circleIcon placeholder-bt CODE_GNB_REBOOT hidden" placeholder="<%=rb.getString("TianJia")%>" @click="addTask">
                    <span class="el-icon el-icon-circle-add"></span>
                </div>
                <!-- 表格组件 -->
                <el-ctable ref="ctable" id="gnb_reboot_list" :url="rebootUrl" :query-params="queryParams">
                    <!-- 列表toolbar -->
                    <template slot="toolbar">
                        <div class='toolbarHeadBtnBoxCls commonQuery'
                            style="height:45px;margin-top: -10px;">
                            <el-query type="normal" @query="query" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
                            <el-date-picker style='margin-left: 20px;' v-model="dateValue"
                                type="datetimerange" value-format="yyyy-MM-dd HH:mm:ss" range-separator="——"
                                @change="dateChange" start-placeholder='<%=rb.getString("KaiShiShiJian")%>'
                                end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
                            </el-date-picker>
                        </div>
                    </template>
                    <!-- 列表columns -->
                    <el-table-column prop="op" label=" " width="50" align="center">
                        <template slot-scope="scope">
                            <div class="el-icon el-icon-operation-more" v-clickoutside="hideMenus"
                                @click="optClick(scope.row,event)"></div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("RenWuMingCheng")%>' min-width="300" prop="TASK_NAME"></el-table-column>
                    <el-table-column label='<%=rb.getString("CaoZuoRen")%>' width="80" prop="CREATE_USER"></el-table-column>
                    <el-table-column label='<%=rb.getString("CaoZuoShiJian")%>' width="200" prop="CREATE_TIME"></el-table-column>
                    <el-table-column label='<%=rb.getString("ChangPinXingHao")%>' width="200" prop="PRODUCT_TYPE"></el-table-column>
                    <el-table-column label='<%=rb.getString("ZhuangTai")%>' width="120" prop="TASK_STATUS">
                        <template slot-scope="scope">
                            <div v-html="taskTableStatus(scope.row.TASK_STATUS)"></div>
                        </template>
                    </el-table-column>
                    <el-table-column label='<%=rb.getString("JinDu")%>' width="100" prop="TASK_PROGRESS"></el-table-column>
                    <el-table-column label='<%=rb.getString("JieGuo")%>' width="100" prop="TASK_RESULT" :formatter="resultFmt"></el-table-column>
                    <el-table-column label='<%=rb.getString("KaiShiShiJian")%>' width="180" prop="START_TIME"></el-table-column>
                    <el-table-column label='<%=rb.getString("JieShuShiJian")%>' prop="END_TIME" width="180"></el-table-column>
                </el-ctable>
            </el-tab-pane>
        </el-tabs>

        <!-- 菜单 -->
        <el-cmenu ref="menu" @click="menuClick" :data="menus"></el-cmenu>

        <el-slide ref="slide" position="bottom" :footer="false" height="300" @cancel="hideSlide"
            title="<%=rb.getString("JieGuo")%>">
            <el-ctable ref="result" id="resultTable" :url="resultUrl" :query-params="resultParams"
                :time="6">
                <el-table-column prop="SERIAL_NUMBER" label="<%=rb.getString("XiaoZhanBianMa")%>"></el-table-column>
                <el-table-column prop="HOST_NAME" label="<%=rb.getString("HostName")%>"></el-table-column>
                <el-table-column prop="PROGRESS_STATUS" label="<%=rb.getString("ZhuangTai")%>">
                    <template slot-scope="scope">
                        <div v-html="resultTableStatus(scope.row.PROGRESS_STATUS)"></div>
                    </template>
                </el-table-column>
                <el-table-column prop="PROGRESS_RESULT" label="<%=rb.getString("JieGuo")%>" :formatter='resultFmt'></el-table-column>
                <el-table-column prop="FAILURE_REASON" label="<%=rb.getString("ShiBaiYuanYin")%>"></el-table-column>
                <el-table-column prop="RUN_TIME" label="<%=rb.getString("ShiJian")%>"></el-table-column>
                <!-- 查询 toolbar -->
                <template slot="toolbar">
                    <el-query type="normal" placeholder="<%=rb.getString("XiaoZhanBianMa")%>" @query="resultQuery"></el-query>
                </template>
            </el-ctable>
        </el-slide>

        <el-slide ref="addTask" title='<%=rb.getString("XinJianRenWu")%>' :subloading="slideSubmitLoading"
            position="top" @cancel="hideAddClose" @ok="saveTask" :url="taskAddURL"
            class='commonBorderSlide'></el-slide>
    </div>

    <script>
        var refreshTaskTimer;
        var gNBRebootVue = new Vue({
            el: '#gnodeb_reboot_ctn',
            data() {
                return {
                    taskAddURL: '',
                    rebootUrl: '${ctx}/task/reboot/getRebootTaskList.action',
                    resultUrl: '',
                    queryForm: {
                        taskName: '',
                        timeRange: []
                    },
                    queryParams: {
                        timeZone: timeZone,
                        likeFields: 'taskName',
                        isGnb: 1,
                        taskName: '',
                        startTime: '',
                        endTime: '',
                    },
                    resultParams: {
                        task_id: '',
                        searchText: '',
                        timeZone: timeZone,
                        productType: '',
                        isGnb: 1
                    },
                    menus: [],
                    dateValue: [],
                    slideSubmitLoading: '',
                }
            },
            watch: {
                dateValue(newVal) {
                    var vm = this;
                    if (!newVal) {
                        newVal = [];
                        vm.queryParams.startTime = '';
                        vm.queryParams.endTime = '';
                    }
                },
            },
            methods: {
                dateChange(val) {
                    var vm = this;
                    vm.dateValue = val;
                    if (val != null) {
                        this.queryParams.startTime = vm.dateValue[0];
                        this.queryParams.endTime = vm.dateValue[1];
                    }
                },
                resultFmt(row, column, cellValue, index) {//软件升级 任务结果fmt
                    var resultObj = {
                        "1": "<%=rb.getString("ChengGong")%>",
                        "2": "<%=rb.getString("BuFenChengGong")%>",
                        "3": "<%=rb.getString("ShiBai")%>",
                        "": ""
                    }
                    return resultObj[cellValue];
                },
                resetQuery() {
                    Object.assign(this.queryForm, {
                        taskName: '',
                        timeRange: []
                    });
                },
                query(text) {
                    var vm = this;

                    vm.resetQuery();
                    Object.assign(vm.queryParams, {
                        taskName: text,
                        startTime: '',
                        endTime: ''
                    });
                },
                advanceQuery() {
                    var vm = this,
                        startTime = '',
                        endTime = '';

                    if (vm.queryForm.timeRange && vm.queryForm.timeRange.length) {
                        startTime = vm.queryForm.timeRange[0];
                        endTime = vm.queryForm.timeRange[1];
                    }

                    vm.queryParams.searchText = '';
                    Object.assign(vm.queryParams, {
                        taskName: vm.queryForm.taskName,
                        startTime: startTime,
                        endTime: endTime
                    });
                },
                resultQuery(text) {
                    var vm = this;

                    vm.resultParams.searchText = text;
                },
                resetResultQuery() {
                    var vm = this;

                    vm.resultParams.searchText = '';
                },
                hideMenus() {
                    this.$refs.menu.hide()
                },
                optClick(row, ev) {
                    var vm = this,
                        taskId = row.TASK_ID,
                        status = row.TASK_STATUS;

                    if (writableMap['CODE_GNB_REBOOT'] == true) {
                        vm.menus = [
                            { label: '<%=rb.getString("JieGuo")%>', code: 'view', taskId: taskId },
                            { label: '<%=rb.getString("KaiShi")%>', code: 'start', taskId: taskId },
                            { label: '<%=rb.getString("ZanTing")%>', code: 'stop', taskId: taskId },
                            { label: '<%=rb.getString("ZhongZhiRenWu")%>', code: 'end', taskId: taskId },
                            { label: '<%=rb.getString("ShanChu")%>', code: 'del', taskId: taskId }
                        ];
                    } else {
                        vm.menus = [
                            { label: '<%=rb.getString("JieGuo")%>', code: 'view', taskId: taskId }
                        ];
                    }

                    initTaskStatus(status, vm.menus);

                    this.$nextTick(function () {
                        document.body.click();
                        vm.showMenus(ev);
                    });
                },
                showMenus(evt) {
                    this.$refs.menu.show(evt)
                },
                menuClick(row) {
                    var vm = this,
                        code = row.code,
                        codes = {
                            view: vm.viewResult,
                            start: vm.activeTask,
                            stop: vm.suspendTask,
                            end: vm.terminateTask,
                            del: vm.delTask
                        };

                    if (codes[code]) codes[code](row);
                },
                viewResult(row) {
                    var vm = this;

                    Object.assign(vm.resultParams, {
                        task_id: row.taskId,
                        searchText: ''
                    });
                    vm.resetResultQuery();
                    vm.resultUrl = '${ctx}/task/reboot/getRebootTaskProgress.action';
                    vm.$refs.slide.showSlide();
                },
                activeTask(row) {
                    var vm = this;

                    axios.post('${ctx}/task/reboot/activeTask.action', stringify({
                        taskId: row.taskId,
                        isGnb: true
                    })).then(function (response) {
                        var data = response.data;
                        if (data["success"]) {
                            vm.$refs.ctable.refresh()
                        } else {
                            vm.$message.error(data["message"])
                        }
                    })
                },
                suspendTask(row) {
                    var vm = this;

                    axios.post('${ctx}/task/reboot/suspendTask.action', stringify({
                        taskId: row.taskId,
                        isGnb: true
                    })).then(function (response) {
                        var data = response.data;
                        if (data["success"]) {
                            vm.$refs.ctable.refresh()
                        } else {
                            vm.$message.error(data["message"])
                        }
                    })
                },
                terminateTask(row) {
                    var vm = this;

                    axios.post('${ctx}/task/reboot/terminateRebootTask.action', stringify({
                        taskId: row.taskId,
                        isGnb: true
                    })).then(function (response) {
                        var data = response.data;
                        if (data["success"]) {
                            vm.$refs.ctable.refresh()
                        } else {
                            vm.$message.error(data["message"])
                        }
                    })
                },
                delTask(row) {
                    var vm = this;

                    vm.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>', QueRen, {
                        customClass: 'warningConfirm',
                        confirmButtonText: '<%=rb.getString("QueDing")%>',
                        cancelButtonText: '<%=rb.getString("QuXiao")%>',
                        type: 'warning',
                        closeOnClickModal: false
                    }).then(() => {
                        axios.post('${ctx}/task/reboot/delRebootTask.action', stringify({
                            taskId: row.taskId,
                            isGnb: true
                        })).then(function (response) {
                            var data = response.data;
                            if (data["success"]) {
                                vm.$refs.ctable.refresh()
                                vm.$message({
                                    type: 'success',
                                    message: '<%=rb.getString("ChengGong")%>'
                                });
                                vm.$refs.slide.hide();
                            } else {
                                vm.$message.error(data["message"])
                            }
                        }).catch(function (error) {

                        })
                    }).catch()
                },
                resultFmt(row, column, cellValue, index) {
                    var code = {
                        1: '<%=rb.getString("ChengGong")%>',
                        2: '<%=rb.getString("ZhongZhi")%>',
                        3: '<%=rb.getString("ShiBai")%>'
                    };

                    if (code[cellValue] != undefined) {
                        return code[cellValue]
                    } else {
                        return ""
                    }
                },
                hideSlide() {
                    this.$refs.slide.hide();
                },
                hideAddClose() {
                    eventBus.$emit('cancel-reboot-task');
                },
                hideAdd() {
                    this.$refs.addTask.hide();
                    this.$refs.ctable.refresh();
                },
                addTask() {
                    var vm = this;
                    vm.slideSubmitLoading = false;
                    vm.$refs.addTask.showSlide();
                    vm.taskAddURL = '${ctx}/gnb/reboot/toRebootTaskAddPage.action';
                },
                saveTask() {
                    eventBus.$emit('save-reboot-task');
                },
                refreshTaskTable() {
                    this.$refs.ctable.refresh();
                }
            },
            mounted() {
                var vm = this;
                eventBus.$off('hide-reboot-task').$on('hide-reboot-task', this.hideAdd);
                clearInterval(refreshTaskTimer);
                refreshTaskTimer = setInterval(function () {
                    var gnbRebootTaskTable = $("#gnodeb_reboot_ctn");
                    if (!gnbRebootTaskTable.length) {
                        clearInterval(refreshTaskTimer);
                        return;
                    }
                    vm.refreshTaskTable();
                }, 6000);
            }
        })
    </script>

</body>

</html>