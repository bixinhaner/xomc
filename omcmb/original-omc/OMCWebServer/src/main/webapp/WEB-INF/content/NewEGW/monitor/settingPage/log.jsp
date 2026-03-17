<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#egwLogsPage{
    width:100%;
    height:calc(100% - 10px);
    position: relative;
}
#egwLogsPage .itemMainBoxCls{
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:8px;
	background:#fff;
	padding:0px 20px;
	height:100%;
}
#egwLogsPage .itemMainBoxTitle {
	height:36px;
	line-height: 36px;
	font-size:14px;
	font-weight:bold;
}
#egwLogsPage .egwLogsTableBoxCls{
    height: 50%;
    border:1px solid #d5dcec;
}
#egwLogsPage .egwLogTaskStatusBoxCls{
    display: flex;
    align-items: center;
    padding-bottom: 50px;
    margin: 20px 0px;
}
#egwLogsPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
    box-sizing: border-box;
    width: calc(100% - 50px);
    position: absolute;
    bottom: 0px;
}
#egwLogsPage  .pro-run{
    background: url('${ctx}/css/images/bi/enb_progress.gif') no-repeat center;
}
</style>

<div id="egwLogsPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">File List</div>
		<div class="egwLogsTableBoxCls">
			<el-ctable ref="egwLogFileTable" :rownumber="true" id="egwLogFileTable" :time="6" :url="logFileTableUrl" :query-params="logFileQueryParams" height="100%" pagination="true">
                <el-table-column label='' width="90" prop="">
                    <template slot-scope="scope">
                        <div class="el-icon el-icon-operation-download" @click="downlodLogFile(scope.row,event)" ></div>
                        <div class="el-icon el-icon-operation-delete CODE_EGW hidden" @click="delCollectFile(scope.row,event)"  ></div>
                    </template>
                </el-table-column>
                <el-table-column label='<%=rb.getString("WenJianMing")%>' min-width="240" prop="file_name"></el-table-column>
                <el-table-column label='<%=rb.getString("GengXinShiJian")%>' min-width="150" prop="upload_time"></el-table-column>
            </el-ctable>
		</div>
        <div class="egwLogTaskStatusBoxCls" v-if="isWritable">
            <span><%=rb.getString("ZhuangTai")%></span> 
            <div v-html='eGWDeviceLogStatus' style='padding-left: 20px; display: flex;align-items: center;'></div>
        </div>
        <div class='itemMainBoxFooter' v-if="isWritable">
            <el-button type="primary" @click="collectLogsClick" style="margin-left:20px;">Collect Logs</el-button>
        </div>
	</div>
    
</div>

<script>
var updateEgwLogTaskDataTimer;
var egwLogsPage = new Vue({
	el: '#egwLogsPage', 
	data() {
		var vm = this;
		return {
            egwCode:'',
            egwSn:'',
			logFileTableUrl:'',
            logFileQueryParams:{
                timeZone:timeZone,
                taskId: '',
            },
            rowTaskData:{},
            eGWDeviceLogStatus: '',
		};
	},
	computed: {
        isWritable() {
            return writableMap.CODE_EGW == true;
        }
    },
	methods: {
        // 初始化
		init(row,code,sn,status){
			var vm =this;
			vm.egwCode = code;
            vm.egwSn = sn;

            vm.refreshLogTaskStatus();
            vm.createEgwLogTaskTimer();
            
		},
        // 请求任务状态
        refreshLogTaskStatus(){
            var vm = this;
                params={
                    timeZone:timeZone,
                    device_code: vm.egwCode,
                    device_type: 'EGW',
                    page:1,
                    rows:50,
                    sort:'',
                    order:''
                },
                urls = '${ctx}/cell/collect/getImmediateCollectLogTaskPageList.action';

            axios.post(urls,stringify(params)).then(function(response){
                let data = response.data;
                if(data.rows.length > 0){
                    vm.rowTaskData = data.rows[0];
                    vm.logFileQueryParams.taskId = data.rows[0].task_id;
                    vm.logFileTableUrl = '${ctx}/cell/collect/getImmediateCollectLogFileDataList.action';
                    
                    vm.rowTaskData.task_status == 0 && (vm.eGWDeviceLogStatus = '<span class="el-icon el-icon-status-waiting1 curStatus"></span><%=rb.getString("DengDai")%>');
                    vm.rowTaskData.task_status == 1 && (vm.eGWDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("JinXingZhong")%>');
                    vm.rowTaskData.task_status == 2 && (vm.eGWDeviceLogStatus = '<span class="el-icon el-icon-status-success greenIcon curStatus"></span><%=rb.getString("ChengGong")%>');
                    vm.rowTaskData.task_status == 3 && (vm.eGWDeviceLogStatus = '<span class="el-icon el-icon-status-failed redIcon curStatus"></span><%=rb.getString("LogsShiBai")%>');
                    vm.rowTaskData.task_status == 4 && (vm.eGWDeviceLogStatus = '<span class="el-icon el-icon-status-terminate curStatus"></span><%=rb.getString("ZhongZhi")%>');
                }
                
            }).catch(function(error){})
        },
        // 创建定时器
        createEgwLogTaskTimer(){
            var vm = this;

            if(updateEgwLogTaskDataTimer){
				clearInterval(updateEgwLogTaskDataTimer);
			}
			updateEgwLogTaskDataTimer = setInterval(function(){    
				var egwLogPageCtn = $("#egwLogsPage");			
				if(!egwLogPageCtn.length) {
					clearInterval(updateEgwLogTaskDataTimer);
					return;
				}
				vm.refreshLogTaskStatus();
			},6000);
        },
		// 下载日志文件
        downlodLogFile(row,ev){
            var vm = this,
                urls = '${ctx}/cell/collect/getDownloadFileNumber.action',
                params = {
                    taskIds: row.task_id,
                    timeZone: timeZone,
                    fileName:row.file_name
                };

            axios.post(urls,stringify(params)).then(function(response){
                var data = response.data;
                if(data.length>0){
                    exportByForm("${ctx}/cell/collect/doDownloadImmediateCollectLogFile.action",params);
                }else{
                    vm.$message.error('<%=rb.getString("WenJianBuCunZai")%>')
                }
            }) 
        },
        /**
		 * 删除任务  -- 设备上报日志 、告警日志 
		 * @param row:当前数据
		*/
	    delCollectFile(row,ev){
	    	var vm = this , 
                url='${ctx}/cell/collect/doClearImmediateCollectLogFile.action' ,
                params = {
                    taskIds: row.task_id,
                    fileName : row.file_name
                };
	    	
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
		    			vm.$refs.egwLogFileTable.refresh()
		    			vm.$message({
			    			type:'success',
			    			message:'<%=rb.getString("ChengGong")%>'
			    		})
		    		}else{
		    			vm.$message.error(data["message"])
		    		}
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()
	    },
        // 日志收集事件
        collectLogsClick(){
            var vm = this,
                params = {
                    timeZone: timeZone,
                    serial_number: vm.egwSn,
                    isReboot: 'false',
                    device_code: vm.egwCode,
                    device_type: 'EGW',
                    execute_type: 'Immediately',
                    reportPeriod: '',
                    start_time: undefined,
                    end_time: undefined
                },
                url = '${ctx}/cell/collect/goImmediateCollectLogFile.action',
                message = '<%=rb.getString("ChengGong")%>';
        
            axios.post(url,stringify(params)).then(function(response){
                var data = response.data;
                if(data["success"]){
                    vm.$message({
                        message: message,
                        type:'success',
                    })
                    vm.$refs.egwLogTable.refresh();//表格刷新
                    vm.refreshLogTaskStatus();//刷新任务状态
                }else if(data.responseCode == "901"){
                    vm.$message.error(data.message);
                }else if(data.responseCode == "401"){//存在未完成的任务时，再次创建任务失败，给出提示
                    vm.$message.error(data["message"]);
                }else{
                    vm.$message.error('<%=rb.getString("ShouJiShiBai")%>');
                }
            })
        },
		
	},
	mounted() {
		eventBus.$off("egw-data").$on("egw-data",this.init)
	}
});

</script>
