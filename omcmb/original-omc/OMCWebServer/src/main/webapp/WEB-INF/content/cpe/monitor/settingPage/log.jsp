<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
.borderPage {
	border:1px solid #d5dcec;
	border-radius:10px;
	height:100%;
	background:#fff;
}
.form-info {
	margin:10px 20px;
}
.form-info .info-title {
	color:#7A7992;
	display:inline-block;
	width:120px;
}
.form-info .el-radio-group .el-radio {
	display:block;
	margin-bottom:20px;
	margin-left:0 !important;
}
.item-title {
	margin:20px 0;
	font-weight:bold;
}
.form-info .el-form-item__label {
	font-size:12px;
	text-align:left;
	line-height:unset;
}
.linkStyle {
	color:#4d84ff;
	font-size:14px;
	text-decoration:underline;
	cursor:pointer;
}
.borderIcon {
	display:inline-block;
	width:26px;
	height:26px;
	line-height:26px;
	text-align:center;
	border:1px solid #d7d7e6;
	border-radius:4px;
}
.wenJianBox{
	height: 60%;
	display: flex;
	overflow: visible;
	border: 1px solid #d1ecf5;
}
.wenJianBoxPad{
	width: 260px;
	border-right: 1px solid #d1ecf5;
}
#cpeLogsPage .el-ctable-toolbar {
	padding:10px 0 !important;
}
#deviceReportLogTable.el-table {
	border:1px solid #d5dcec;
}
.pro-run{
	background: url('${ctx}/css/images/bi/enb_progress.gif') no-repeat center;
}
</style>

<div id="cpeLogsPage" class="borderPage">
	
	<div class="form-info">
		<el-ctable ref="deviceReportLogTable" :rownumber="true" id="deviceReportLogTable" :time="6" :url="reportLogTableUrl" :query-params="queryParams_reportLog" height="400px" pagination="true">
			<template slot="toolbar">
				<div style="padding:0 20px;">
					<span class="titleClass"> File List</span>
					<div style="float:right;">
						<el-button icon="el-icon el-icon-operation-download" @click="downlodFile(rowTaskData,'device')"><%=rb.getString("XiaZai")%></el-button>
						<el-button v-if="isWritable" icon="el-icon el-icon-operation-delete" @click="delCollectFile(rowTaskData,'device')"><%=rb.getString("ShanChu")%></el-button>
					</div>
				</div>
			</template>
			
			<el-table-column label='' width="90" prop="">
				<template slot-scope="scope">
					<div class="el-icon el-icon-operation-view" @click="viewLogFile(scope.row,event)"  ></div>
	            	<div class="el-icon el-icon-operation-download" @click="downlodFile(scope.row,'file')" ></div>
	            	<div class="el-icon el-icon-operation-delete" @click="delCollectFile(scope.row,'file')"  ></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("WenJianMing")%>' min-width="150" prop="file_name"></el-table-column>
			<el-table-column label='<%=rb.getString("GengXinShiJian")%>' min-width="150" prop="upload_time"></el-table-column>
		</el-ctable>
	</div>
	
	<div  class="form-info commonFlex" style='padding-bottom: 50px; ' v-if="isWritable">
		<span><%=rb.getString("ZhuangTai")%></span> 
		<div v-html='cpeDeviceLogStatus' style='padding-left: 20px;'></div>
	</div>
	
	<div class="form-info" v-if="isWritable">
		<el-button type="primary" @click="collectLog">Collect Log</el-button> 
		<span v-if="false" @click="viewLogTask" class="linkStyle" style="margin-left:10px;">View Logs Tasks</span>
	</div>
		
	<el-dialog :visible.sync="dialogVisible" title="<%=rb.getString("WenJianXinXi")%>" append-to-body>
	  	<div class="wenJianBox">
			<div class="wenJianBoxPad" style="">
				<el-ctable id="logFileList" ref="fileList" :url="logFileListUrl" :height="'100%'" @row-click="fileSelectChange" :pagination="false"
						:query-params="params_filelist">
					<!-- 主列表 -->
					<el-table-column label='<%=rb.getString("WenJianLieBiao")%>' prop="un_file_name" ></el-table-column>
				</el-ctable>
			</div>
			<div style="width: 100%;padding: 10px;">
				<textarea id="immediateCollectFileContent" class="border border-box" style="padding-left:10px;border-style: none;width: 100%;height: 97%;resize: none;"></textarea>
			</div>
		</div>
	 </el-dialog>
	<el-dialog title="<%=rb.getString("WenJianXinXi")%>"  style="width:100%;" :close-on-click-modal="false" @close="cancelDialog">
		<div class="wenJianBox">
			<div class="wenJianBoxPad" style="">
				<el-ctable id="logFileList" ref="fileList" :url="logFileListUrl" :height="'100%'" @row-click="fileSelectChange" :pagination="false"
						:query-params="params_filelist">
					<!-- 主列表 -->
					<el-table-column label='<%=rb.getString("WenJianLieBiao")%>' prop="un_file_name" ></el-table-column>
				</el-ctable>
			</div>
			<div style="width: 100%;padding: 10px;">
				<textarea id="immediateCollectFileContent" class="border border-box" style="padding-left:10px;border-style: none;width: 100%;height: 97%;resize: none;"></textarea>
			</div>
		</div>
	</el-dialog>
	
	 <form id = "exportEventLog" style="display:none" method="post"></form>
	 
</div>
<script>
var cpeLogsPage = new Vue({
	el: '#cpeLogsPage', 
	data() {
		
		return {
			reportLogTaskUrl:'${ctx}/cell/collect/getImmediateCollectLogTaskPageList.action',
			queryParams_reportTask:{
				device_type:'CPE',
				device_code:'',
				search_text:'',			
				like_fields:'serial_number',
				timeZone:timeZone,
				serial_number:'',
				cpe_name:'',
				mac_address:'',
				imsi:'',
				start_time:'',
				end_time:'',
				page:1,
				rows:50,
				sort:'',
				order:''
			},
			
			reportLogTableUrl:'',
			queryParams_reportLog:{
				timeZone: timeZone,
				taskId: "",
				deviceType:'CPE'
			},
			
			dialogVisible: false,
			params_filelist: {
				taskId: '',
				fileName: '',
				fileType: ''
			},
			logFileListUrl: '${ctx}/cell/collect/doUnZipImmedLogFile.action',
			rowTaskData:[],
			rowData:[],
		    rowDataFile:[],
		    
		    code:'',
		    cpeDeviceLogStatus: '',
		};
	},
	computed: {
        isWritable() {
            return writableMap.CODE_CPE_LOGS == true;
        }
	},
	watch:{
		
	},
	methods: {
		init(code,mac,sn,status){
			var vm = this;
			vm.code = code;
			
			vm.queryParams_reportTask.mac_address = mac;
			
			if(window.cpeUpdateRowDataTimer){
				clearInterval(window.cpeUpdateRowDataTimer);
			}
			window.cpeUpdateRowDataTimer = setInterval(function(){    
				var cpeLogPageCtn = $("#cpeLogsPage");			
				if(!cpeLogPageCtn.length) {
					clearInterval(window.cpeUpdateRowDataTimer);
					return;
				}

				axios.post(vm.reportLogTaskUrl,stringify(vm.queryParams_reportTask)).then(function(response){
					let data = response.data;
					
					if(data.rows.length > 0){
						vm.rowTaskData = data.rows[0];
						vm.queryParams_reportLog.taskId = data.rows[0].task_id;
						vm.reportLogTableUrl = '${ctx}/cell/collect/getImmediateCollectLogFileDataList.action'
						
						if(vm.rowTaskData.execute_type == 'Immediately'){
							vm.rowTaskData.task_status == 0 && (vm.cpeDeviceLogStatus = '<span class="el-icon el-icon-status-waiting1 curStatus"></span><%=rb.getString("DengDai")%>');
							vm.rowTaskData.task_status == 1 && (vm.cpeDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("JinXingZhong")%>');
							vm.rowTaskData.task_status == 2 && (vm.cpeDeviceLogStatus = '<span class="el-icon el-icon-status-success greenIcon curStatus"></span><%=rb.getString("ChengGong")%>');
							vm.rowTaskData.task_status == 3 && (vm.cpeDeviceLogStatus = '<span class="el-icon el-icon-status-failed redIcon curStatus"></span><%=rb.getString("LogsShiBai")%>');
							vm.rowTaskData.task_status == 4 && (vm.cpeDeviceLogStatus = '<span class="el-icon el-icon-status-terminate curStatus"></span><%=rb.getString("ZhongZhi")%>');
						}else{
							vm.rowTaskData.task_status == 0 && (vm.cpeDeviceLogStatus = '<span class="el-icon el-icon-status-waiting1 curStatus""></span><%=rb.getString("DengDai")%>');
							vm.rowTaskData.task_status == 1 && (vm.cpeDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("JinXingZhong")%>');
							vm.rowTaskData.task_status == 2 && (vm.cpeDeviceLogStatus = '<span class="el-icon el-icon-status-success greenIcon curStatus"></span><%=rb.getString("ChengGong")%>');
							vm.rowTaskData.task_status == 3 && (vm.cpeDeviceLogStatus = '<span class="el-icon el-icon-status-failed redIcon curStatus"></span><%=rb.getString("LogsShiBai")%>');
							vm.rowTaskData.task_status == 4 && (vm.cpeDeviceLogStatus = '<span class="el-icon el-icon-status-terminate curStatus"></span><%=rb.getString("ZhongZhi")%>');
							vm.rowTaskData.task_status == 5 && (vm.cpeDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("ZhongZhiZhong")%>');
							vm.rowTaskData.task_status == 6 && (vm.cpeDeviceLogStatus = '<span class="el-icon el-icon-status-terminate curStatus"></span><%=rb.getString("ZhongZhi")%>');
							vm.rowTaskData.task_status == 7 && (vm.cpeDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("JinXingZhong")%>');
							vm.rowTaskData.task_status == 8 && (vm.cpeDeviceLogStatus = '<span class="el-icon el-icon-status-waiting1 curStatus"></span<span><%=rb.getString("ZhouQiShangBaoSheZhiChengGongCQSX")%></span>');
						}
					}
				}).catch(function(error){})
			},6000);
			
		},
		// 日志收集事件
		collectLog(){
			var vm = this,
	            urls="${ctx}/cell/collect/goImmediateCollectLogFile.action"
	            params = {
	                timeZone : timeZone,
	                isReboot:false,
	                device_type : 'CPE',
	                device_code : vm.code,
	                execute_type: 'Immediately',
	                start_time: '',
	            };
	        axios.post(urls,stringify(params)).then(function(response){
	            var data = response.data;
	            if(data["success"]){
	                vm.$message({
	                    type: 'success',
	                    message: '<%=rb.getString("CPERiZhiZhengZaiShouJi")%>'
	                });
					vm.init(vm.code, vm.queryParams_reportTask.mac_address);
	            }else{
	                vm.$message.error(data["message"])
	            }       			    		
	        })
		},
		/**
		 *  查看操作
		 * @param row:当前数据
		*/
	    viewLogFile(row,ev){ 
	    	var vm = this;
	    	
	    	var params={
    			taskId: row.task_id,
				fileName: row.file_name, 
				fileType: 'CPE'
	    	};

	    	Object.assign(vm.params_filelist, params);
	    	$("#immediateCollectFileContent").val("");
	    	vm.dialogVisible = true;
	    	vm.$root.rowData = row;

	    },
	    fileSelectChange(row){
			if(row){
				this.viewImmediateLogFileContent(row);
			}
	    },
		viewImmediateLogFileContent(rowData) {
	    	var vm = this,
			    fileType = 'CPE';
	    	$("#immediateCollectFileContent").val("");
	    	
	        if(rowData.file_name == '' || rowData.file_name == null || rowData.un_file_path == '' || rowData.un_file_path == null){
	    		return;
	    	}else{
	    		var params = {
		   			fileName:rowData.file_name,
		   			unFilePath:rowData.un_file_path,
		   			fileType: fileType
		    	};
	    		$.messager.progress({
		            title : "<%=rb.getString("QingDengDai")%>",
		            text : "<%=rb.getString("JieXiZhong")%>"
		        });
				 
		    	$.post("${ctx}/cell/collect/viewUnZipImmedLogFile.action", params, function (data) {
		    		$.messager.progress("close");
		    		if (data.success) {
		    			if(data.message==""){
		    				showMsg('prompt_msg','<%=rb.getString("WenJianNeiRongWeiKong")%>');
		    			}else{
		    				$("#immediateCollectFileContent").val(data.message);
		    			}
		           } else {
		          		showMsg('prompt_msg','<%=rb.getString("WenJianBuCunZai")%>');
		          	    return;
		           }
		        }, "json");
    		}
	    },
	  //关闭结果->查看页面
    	cancelDialog(){
    		var vm = this;    		
    		$("#immediateCollectFileContent").val("");
			vm.$refs.fileList.refresh();
			vm.$refs.fileList.setCurrentRow();
	    	vm.dialogVisible = false;
    	},
    	/**
		 *  下载文件
		 * @param row:当前数据
		*/
	    downlodFile(row,type){ 
    		var vm = this;
				url='${ctx}/cell/collect/doDownloadImmediateCollectLogFile.action',
	    		params = {
					timeZone: timeZone,
					taskIds:'',
					fileName: '',
					fileType: 'CPE'
				};
			if(type == 'device'){
				params.taskIds = row.task_id;
				if (row.file_num == 0) {
					showMsg('prompt_msg','<%=rb.getString("MeiYouYaoXiaZaiDeWenJian")%>');
					return;
	        	}  
			}else if(type == 'file'){
				params.taskIds = row.task_id;
				params.fileName = row.file_name;
				if (row.file_name == null || !row.file_name || row.file_name == undefined) {
					showMsg('prompt_msg','<%=rb.getString("MeiYouYaoXiaZaiDeWenJian")%>');
					return;
	        	}  
			}
	       	        
			axios.post('${ctx}/cell/collect/getDownloadFileNumber.action',stringify(params)).then(function(response){
	    		var data = response.data;
				if (data.length > 0 || data.fileNum > 0) {
		    		vm.createForm(url,params);
		        } else {
		       	 	vm.$message.error('<%=rb.getString("WenJianBuCunZai")%>') //错误提示信息
		        }
	    	}) 
   
	    },
		/**
		 * 跳转from表格
		 * @param url:地址
		 * @param params：跳转参数
		*/
		createForm(url,param) {
			var body = document.querySelector('body'),
				form = document.createElement('form'),
				params = param || {};
			
			form.style.display = 'none';
			form.action = url;
			form.method = 'post';
			
			if(params) {
				params.token = omctoken;
				for(var key in params) {
					var input = document.createElement('input');
					input.value = params[key];
					input.setAttribute('name',key);
					form.appendChild(input);
				}
			}
			
			body.appendChild(form);
			form.submit();
			form.remove();
		},
		/**
		 * 删除任务  -- 设备上报日志 、告警日志 
		 * @param row:当前数据
		*/
	    delCollectFile(row,type){
	    	var vm = this, 
				url='${ctx}/cell/collect/doClearImmediateCollectLogFile.action',
				params={
					taskIds:'',
					fileName:'',
					deviceType:'CPE'
				};
			
			if(type == 'device'){
				params.taskIds = row.task_id;
			}else if(type == 'file'){
				params.taskIds = row.task_id;
				params.fileName = row.file_name;
			}

	    	vm.$confirm('<%=rb.getString("QueRenShanChuWenJian")%>',QueRen,{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post(url,stringify(params)).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			vm.$refs.deviceListTable.refresh()
		    			vm.$message({
			    			type:'success',
			    			message:'<%=rb.getString("ChengGong")%>'
			    		})
			    		vm.$refs.logFileTable.refresh();
		    		}else{
		    			vm.$message.error(data["message"])
		    		}
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()
	    },
	},
	mounted() {
		eventBus.$off("cpe-data").$on("cpe-data",this.init)
	}
});

</script>
