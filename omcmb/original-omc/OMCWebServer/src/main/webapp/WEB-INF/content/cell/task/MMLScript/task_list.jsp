<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.task-list-wrap {
		flex:1;
		overflow: auto;
	}
	.task-results-wrap {
		flex:1;
		margin-top:10px;
		overflow:auto; 
		border: 1px solid #D5DCEC; 
		border-radius: 8px;
	}
	.toolbar-nopadding .el-ctable-toolbar {
		padding: 0 !important;
	}
</style>
<!-- MML脚本任务 -->
<div id="mml_task_list_ctn" class="panelDefault" style="border:none;display: flex;flex-direction: column;background-color: rgb(246, 247, 251);">
	<div class='task-list-wrap commonTableBorder'>
		<el-ctable class="toolbar-nopadding" id="mml_task_table_ctn"
			ref="taskList"
			:url="taskUrl"
			:query-params="queryParams"
			row-key="TASK_ID"
			:pagination="true"
			:rownumber="true"
			:time="6"
		>
			<template slot="toolbar">
				<div class='toolbarHeadBtnBoxCls commonQuery' style="height:45px;">
					<div @click="toAdd" class="newIconBoxCls-bt CODE_ENB_MML hidden" style="right:15px;top:5px;" tip="<%=rb.getString("TianJia")%>">
						<span class='el-icon el-icon-circle-add'></span>
					</div>
					<el-query type="normal"  @query="query" placeholder='<%=rb.getString("RenWuMingCheng")%>'></el-query>

					<el-date-picker style="width: 300px;margin-left: 10px;"
						v-model="dateRange"
						type="datetimerange"
						start-placeholder='<%=rb.getString("KaiShiShiJian")%>'
						end-placeholder='<%=rb.getString("JieShuShiJian")%>'
						value-format="yyyy-MM-dd HH:mm:ss"
						:picker-options="pickerOpts"
						@change="dateTimeChange"
						size="mini"
					></el-date-picker>
				</div>
			</template>
			<el-table-column label="" width="70">
				<template slot-scope="scope">
					<span class="el-icon el-icon-operation-result" @click="showResult(scope.row)"></span>
					<span class="el-icon el-icon-operation-more-circle" style="margin-left: 5px;" @click="optClick(scope.row, event)" v-clickoutside="handerClose"></span>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("RenWuMingCheng")%>' prop="TASK_NAME"></el-table-column>
			<el-table-column label='<%=rb.getString("ChuangJianZhe")%>' prop="CREATE_USER"></el-table-column>
			<el-table-column label='<%=rb.getString("ChuangJianShiJian")%>' prop="CREATE_TIME"></el-table-column>
			<el-table-column label='<%=rb.getString("Type")%>' prop="CREATE_STATUS" :formatter="typeFmt"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="TASK_STATUS">
				<template slot-scope="scope">
					<div v-html="tastStatusFmt(scope.row, scope.row['TASK_STATUS'], scope.$index)" :key="scope.row.TASK_ID"></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("JinDu")%>' prop="TASK_PROGRESS"></el-table-column>
			<el-table-column label='<%=rb.getString("JieGuo")%>' prop="TASK_RESULT" :formatter="taskResultFmt"></el-table-column>
			<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' prop="START_TIME" min-width="140"></el-table-column>
			<el-table-column label='<%=rb.getString("JieShuShiJian")%>' prop="END_TIME" min-width="140"></el-table-column>
		</el-ctable>

		<el-cmenu ref="menuTask" :data="taskMenus" @click="menuClick"></el-cmenu>
	</div>
	
	<div v-show="resultShow" class='task-results-wrap'>
		<el-ctable class="toolbar-nopadding" id="mml_result_table_ctn"
			:url="resultUrl"
			:query-params="resultQuery"
			:pagination="true"
			:rownumber="true"
			:time="6"
			row-key="SERIAL_NUMBER"
		>
			<template slot="toolbar">
				<div class='toolbarHeadBtnBoxCls commonQuery'>
					<div @click="exportResult" class="newIconBoxCls-bt" style="right:50px;top:5px;" tip="<%=rb.getString("DaoChu")%>">
						<span class='el-icon el-icon-circle-export'></span>
					</div>
					<div @click="closeResult" class="newIconBoxCls-bt" style="right:15px;top:5px;" tip="<%=rb.getString("GuanBi")%>">
						<span class='el-icon el-icon-circle-close'></span>
					</div>
					<span style="display: inline-block; font-size: 14px;font-weight: bold;padding: 0 0 0 15px;">
						<%=rb.getString("JieGuo")%> (
							<span style="color: #4D84FF;">{{taskName}}</span>
						)
					</span>
					<el-query type="normal" @query="queryResult" placeholder='<%=rb.getString("QingShuRuJiZhanChaXunNeiRong")%>'></el-query>
				</div>
			</template>
			<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="SERIAL_NUMBER" width="160"></el-table-column>
			<el-table-column label='<%=rb.getString("HostName")%>' prop="HOST_NAME"></el-table-column>
			<el-table-column label='<%=rb.getString("CLIFile")%>' prop="MML"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="PROGRESS_STATUS">
				<template slot-scope="scope">
					<div v-html="statusFmt(scope.row, scope.row['PROGRESS_STATUS'], scope.$index)" :key="scope.row.SERIAL_NUMBER"></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("JieGuo")%>' prop="PROGRESS_RESULT" :formatter="resultFmt"></el-table-column>
			<el-table-column label='<%=rb.getString("PCILOCKShiBaiYuanYin")%>' prop="FAILURE_REASON"></el-table-column>
			<el-table-column label='<%=rb.getString("XiangQing")%>' prop="DETAIL">
				<template slot-scope="scope">
					<div v-html="detailFmt(scope.row, scope.row['DETAIL'], scope.$index)"></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' prop="RUN_TIME" min-width="140"></el-table-column>
			<el-table-column label='<%=rb.getString("JieShuShiJian")%>' prop="END_TIME" min-width="140"></el-table-column>
		</el-ctable>
	</div>

	<!-- add -->
	<div id="winAddMMLScriptTask" class="slidebarPanel" style="overflow: auto;top: 85px !important;border-radius: 10px;"></div>

	<div id="task_details" class="easyui-dialog" data-options="closed:true" title="<%=rb.getString("XiangQing")%>"></div>
</div>

<script type="text/javascript">
new Vue({
	el: '#mml_task_list_ctn',
	data() {

		return {
			taskName: '',
			taskId: '',
			taskMenus: [],

			taskUrl: '${ctx}/task/MMLScript/getMMLScriptTaskList.action',
			queryParams: {
				timeZone: timeZone,
				likeFields: 'task_name',
				searchText: '',
				startTime: '',
				endTime: ''
			},

			resultUrl: '',
			resultShow: false,
			resultQuery: {
				timeZone: timeZone,
				searchText: ''
			},
			dateRange: [],
			pickerOpts: {
				disabledDate(time) {
					// 禁止选择未来时间 （可以选今天）
					return time.getTime() > Date.now();
				}
			}
		}
	},
	methods: {
		init() {
			var vm = this;

		},
		query(text) {
			this.queryParams.searchText = text.trim();
		},
		queryResult(text) {
			this.resultQuery.searchText = text.trim();
		},
		exportResult() {
			var vm = this;
			var url = "${ctx}/task/MMLScript/exportMMLProgResultToCSV.action";
			
			exportByForm(url, {
				taskId: vm.taskId,
				timeZone: timeZone,
				searchText: vm.resultQuery.searchText
			});
		},
		showResult(row) {
			var vm = this;

			vm.taskName = row.TASK_NAME;
			vm.taskId = row.TASK_ID;
			vm.resultUrl = '${ctx}/task/MMLScript/getMMLScriptTaskProgress.action?task_id=' + row.TASK_ID;
			vm.resultShow = true;
		},
		closeResult() {
			var vm = this;

			vm.resultShow = false;
		},
		statusFmt(row, value, idx) {
			return resultTableStatus(value, row, idx);
		},
		resultFmt(row, col, value, idx) {
			return resultTableResult(value, row, idx);
		},
		detailFmt(row, value, idx) {
			var details = row.DETAIL,
				content = '',
				titleStr = '',
				strList = [];
			
			if(details) {
				if(details.substr(0,2).indexOf('[') < 0) {
					details = '['+ details +']';
				}
				
				var list = eval('('+details+')');
				
				titleStr = '[<br>' + getProps(list, 0) + '<br>]';

				content = details.length > 25? details.substr(0, 20)+'...':details;
			}
			
			var str = [
					'<span href="#" class="lst-note" msg="'+titleStr+'" onclick="getDetail(this)" style="cursor: pointer;color: blue;">',
						content,
					'</span>'
				].join(' ');
			
			return str;
		},

		typeFmt(row, col, value, idx) {
			var status = {
					active: '<%=rb.getString("LiJiZhiXing")%>',
					suspend: '<%=rb.getString("GuaQi")%>',
					timing: '<%=rb.getString("DingShiZhiXing")%>',
					period: '<%=rb.getString("ZhouQiRenWu")%>'
				};

			return status[value];
		},
		tastStatusFmt(row, value, idx) {
			return taskTableStatus(value, row, idx);
		},
		taskResultFmt(row, col, value, idx) {
			return taskTableResult(value, row, idx);
		},
	    optClick(row, ev){
	    	var vm = this;
   	    	var status = row.TASK_STATUS;
   		    var codeName = '';
   		    
	    	vm.taskMenus = [
				{label:'<%=rb.getString("XinXi")%>', cls: 'el-icon el-icon-operation-info', code:'info'},
		    ];

			if(writableMap['CODE_ENB_MML'] == true) {
				vm.taskMenus = [
					{row: row, label:'<%=rb.getString("XinXi")%>', cls: 'el-icon el-icon-operation-info', code:'info'},
					{row: row, label:'<%=rb.getString("KaiShi")%>', cls: 'el-icon el-icon-operation-start', code:'start'},
					{row: row, label:'<%=rb.getString("ZanTing")%>', cls: 'el-icon el-icon-operation-awaiting', code:'wait'},
					{row: row, label:'<%=rb.getString("ZhongZhiRenWu")%>', cls: 'el-icon el-icon-operation-terminate', code:'end'},
					{row: row, label:'<%=rb.getString("ShanChu")%>', cls: 'el-icon el-icon-operation-delete', code:'del'},
				];
			}

			initTaskStatus(status, vm.taskMenus);
	    	
	    	vm.$nextTick(function(){
	    		document.body.click();
   		    	vm.$refs.menuTask.show(ev);
	    	});
	    },
		menuClick(row) {
			var vm = this,
				actions = {
					info: vm.viewTask,
					start: vm.startTask,
					wait: vm.suspendTask,
					end: vm.terminateTask,
					del: vm.deleteTask
				};

			if(actions[row.code]) {
				actions[row.code](row.row);
			}
		},
		viewTask(row) {
			var vm = this;

			var paramArr = ['info', row.TASK_ID];
    		sessionStorage.setItem('addOrInfoFlag', paramArr);
			enbmmlVue.toAddTask('view');
		},
		startTask(row) {
			var vm = this,
				params = {
					taskId: row.TASK_ID
				},
				url = '${ctx}/task/MMLScript/activeTask.action';

        	axios.post(url, stringify(params)).then(function(res) {
				var data = res.data;

        		if (data["success"]) {
        			vm.$refs.taskList.refresh();
        		} else {
					vm.$message.error(data["message"])
        		}
        	});

		},
		suspendTask(row) {
			var vm = this,
				params = {
					taskId: row.TASK_ID
				},
				url = '${ctx}/task/MMLScript/suspendTask.action';

			axios.post(url, stringify(params)).then(function(res) {
				var data = res.data;

        		if (data["success"]) {
        			vm.$refs.taskList.refresh();
        		} else {
					vm.$message.error(data["message"])
        		}
        	});
		},
		terminateTask(row) {
			var vm = this,
				params = {
					taskId: row.TASK_ID
				},
				url = '${ctx}/task/MMLScript/terminateMMLScriptTask.action';

			axios.post(url, stringify(params)).then(function(res) {
				var data = res.data;

        		if (data["success"]) {
        			vm.$refs.taskList.refresh();
        		} else {
					vm.$message.error(data["message"])
        		}
        	});
		},
		deleteTask(row) {
			var vm = this,
				params = {
					taskId: row.TASK_ID
				},
				confirmStr = '<%=rb.getString("QueRenShanChu")%>',
				url = '${ctx}/task/MMLScript/delMMLScriptTask.action';

			vm.$confirm(confirmStr, '<%=rb.getString("QueRen")%>',{
				customClass: "warningConfirm",
				confirmButtonText: '<%=rb.getString("QueDing")%>',
				cancelButtonText: '<%=rb.getString("QuXiao")%>',
				type: 'warning',
				closeOnClickModal: false
			}).then(() => {
				axios.post(url, stringify(params)).then(function(res) {
					var data = res.data;

					if (data["success"]) {
						vm.$refs.taskList.refresh();
					} else {
						vm.$message.error(data["message"])
					}
				});
			}).catch(() => {})
			
		},
		handerClose() {
			var vm = this;

			vm.$refs.menuTask.hide();
		},
		toAdd() {
			var vm = this;

			var paramArr = ['add', {}]
    		sessionStorage.setItem('addOrInfoFlag', paramArr);
			enbmmlVue.toAddTask();
		},
		dateTimeChange(val) {
			var vm = this;

			if(val) {
				vm.queryParams.startTime = val[0];
				vm.queryParams.endTime = val[1];
			}else {
				vm.queryParams.startTime = '';
				vm.queryParams.endTime = '';
			}
		}
	},
	mounted() {
		this.init();
	}
})

function getDetail(el) {
	let titleStr = $(el).attr('msg');
	
	$('#task_details').dialog({
		width: 600,
		height: 500,
		top: 100,
		content: titleStr,
		modal: true,
		closed: false
	});
}
function getProps(list, n) {
	var prepad = '&nbsp;&nbsp;&nbsp;&nbsp;',
		propList = [],
		nextN = n +1;
	
	for(var i = 0; i < n; i++) {
		prepad += '&nbsp;&nbsp;&nbsp;&nbsp;';
	}

	list.map(function(item, idx){
		if(idx) propList.push(prepad + '{');
		else propList.push('&nbsp;&nbsp;&nbsp;&nbsp;' + '{');
		
		for(key in item) {
			
			if(Array.isArray(item[key])) {
				propList.push(prepad +'&nbsp;&nbsp;&nbsp;&nbsp;'+ key + ': [');
				
				propList.push(prepad + getProps(item[key], nextN));
				
				propList.push(prepad +'&nbsp;&nbsp;&nbsp;&nbsp;'+ '],');
			}else if(typeof(item[key]) == 'object') {
				propList.push(prepad + prepad + key + ': {');
				for(mkey in item[key]) {
					propList.push(prepad + prepad +'&nbsp;&nbsp;&nbsp;&nbsp;'+ mkey + ': &quot;' + item[key][mkey] + '&quot;');
				}
				propList.push(prepad + prepad + '},');
			}else {
				propList.push(prepad +'&nbsp;&nbsp;&nbsp;&nbsp;'+ key + ': &quot;' + item[key] + '&quot;');
			}
		}
		
		propList.push(prepad + '},');
	});
	
	return propList.join('<br>');
}
</script>