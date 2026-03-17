<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>


<div id="gnbSettingLogsPage" class='commonWarp'>
	<el-ctable ref="ctableLog"  :url="logUrl" :query-params="queryParams" pagination="true" :rownumber=true :row-key="'serial_number'">
        <template slot="toolbar">
         	<div class='toolbarHeadBtnBoxCls'>
         		<div class='commonGeneral12' style='margin-left: 20px;'><%=rb.getString("WenJian")%></div>            	
             </div>             
        </template>
        <el-table-column width="40">
             <template slot-scope="scope">
                 <div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose"></div>
             </template>
        </el-table-column>                     
        <el-table-column prop="serial_number" label="<%=rb.getString("JiZhanXuLieHao")%>"></el-table-column>
		<el-table-column prop="task_status" label="<%=rb.getString("ShouJiZhuangTai")%>" width="180">
			<template slot-scope="scope">
				<div v-if="scope.row.execute_type == 'Immediately'">						
					<div v-if="scope.row.task_status == 0">
						<span class="el-icon el-icon-status-waiting1 curStatus"></span>
						<span><%=rb.getString("DengDai")%></span>
					</div>
					<div v-if="scope.row.task_status == 1">
						<span class="el-icon el-icon-status-inProgress curStatus"></span>
						<span><%=rb.getString("JinXingZhong")%></span>
					</div>
					<div v-if="scope.row.task_status == 2">
						<span class="el-icon el-icon-status-success curStatus"></span>
						<span><%=rb.getString("ChengGong")%></span>
					</div>
					<div v-if="scope.row.task_status == 3">
						<span class="el-icon el-icon-status-failed curStatus"></span>
						<span><%=rb.getString("LogsShiBai")%></span>
					</div>
					<div v-if="scope.row.task_status == 4">
						<span class="el-icon el-icon-status-terminate curStatus"></span>
						<span><%=rb.getString("ZhongZhi")%></span>
					</div>
				</div>
			</template>
		</el-table-column>
		<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failureReason" show-overflow-tooltip="true"></el-table-column>
		<el-table-column prop="update_time" label="<%=rb.getString("LogsGengXinShiJian")%>"></el-table-column>                          
	</el-ctable>
         
	<el-cmenu ref="menus" :data="menus" @click="clickMenu"></el-cmenu>	
</div>

<script type="text/javascript">
	var curSnCellCode = gnbTabSettingVue.rowData.small_cell_code;
	var gnbSettingLogsPageVue = new Vue({
	    el: '#gnbSettingLogsPage',
	    data() {
	    	return {
	    		menus: [],
	    		queryParams: {
	    			isGnb: 1,
					search_text: '',
					device_type: 'eNB',
					device_code: curSnCellCode,
					like_fields: 'serial_number',
					timeZone: timeZone,
					start_time: '',
					end_time: ''
	            }, 
				logUrl: '${ctx}/cell/collect/getImmediateCollectLogTaskPageList.action',
				/*tableData: [
					{
						'device_code':'35457454545006',
	            		'device_type':'eNB',
	            		'enb_time':'2023-01-17 00:00:00',
	            		'execute_type':'Immediately',
	            		'failureReason':'',
	            		'report_period':'',
	            		'serial_number':'12000000000066',
	            		'start_time':'2023-01-17 00:00:00',
	            		'task_id':'hhjghjgj001',
	            		'tsak_status':7,
	            		'update_time':'2023-01-17 00:00:00',
					}
				],*/
				rowData:[],
	    	}
	    },
	    methods: {
	    	/*init(){
	    		var vm = this;
	    		//vm.queryParams.device_code = gnbTabSettingVue.rowData.small_cell_code;
	    		//console.log(gnbTabSettingVue.rowData.small_cell_code)
	    	},*/
			/**
	      	 * 表格数据点击出现menus菜单事件
	        * @param row{object}   行数据
	        * @param ev{object}   event数据
	        */
	        optClick(row,ev) {
	            var vm = this, status = row.task_status, terminateFlag = false , showFlag = true , delFlag = false;
				
	            vm.rowData = row;
				if(status == 0 || status == 1 || status == 7 || status == 8){
				    terminateFlag = true;
				}else{
				    terminateFlag = false;
				}

		          //进行中 - 表格操作-删除为置灰状态
	           if(status == 1 || status == 5 || status == 7){
	               delFlag = false;
	           }else{
	               delFlag = true;
	           }

		       vm.menus= [
	                {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:"el-icon el-icon-operation-terminate CODE_GNB_LOGS hidden" ,code:'terminate',disable:!terminateFlag ,show:showFlag, row: row},
	                {label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download" ,code:'download', row: row},
	                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_GNB_LOGS hidden",code:'del',disable:!delFlag, row: row}
	           ];
	            
	           vm.$nextTick(function(){
	               document.body.click();
	               vm.$refs.menus.show(ev);
	           });
	        },
	        
	        /**
			 *  获取点击项数据
			 * @parame ev:点击属性数据
			*/
		    clickMenu(ev){
				var vm = this,
		    		codes = {	
			    		terminate: vm.terminateCollectTask,
			    		download: vm.downlodFile,
			    		del: vm.delCollectFile
			    	}
		    	if(codes[ev.code]){
		    		codes[ev.code](vm.rowData)
		    	}
		    },
		    terminateCollectTask(row) {
				var vm = this , 
					url = '${ctx}/cell/collect/goTerminateImmediateCollectLogFile.action',
					status = row.task_status,
					params = {
						taskIds: row.task_id,
						device_code: row.device_code,
						execute_type: row.execute_type,
						isGnb: 1
					};

				if (status != 0 && status != 1 && status != 7 && status != 8) {
					showMsg('prompt_msg','<%=rb.getString("MeiYouKeTingZhiShouJiDeSheBei")%>');
					return;
				}
				
				axios.post(url, stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
					vm.$refs.ctableLog.refresh()
					vm.$message({
						type:'success',
						message:'<%=rb.getString("ChengGong")%>'
					})
					}else{
					vm.$message.error(data["message"])
					}
				}).catch(function(error){});
			},
		    //下载
		    downlodFile(row){   	
				var vm = this,
					taskId = row.task_id,
					fileNum = row.file_num,
					fileName = row.file_name||'',
					params = {
						taskIds: taskId,
						timeZone: timeZone,
						fileName: fileName,
						isGnb: 1
					},
					checkURL = '${ctx}/cell/collect/getDownloadFileNumber.action',
					url = "${ctx}/cell/collect/doDownloadImmediateCollectLogFile.action";
				
				axios.post(checkURL,stringify(params)).then(function(response){
					var data = response.data;
					if(data.length > 0 || data.fileNum > 0){
						vm.createForm(url, params);
					}else{
						vm.$message.error('<%=rb.getString("WenJianBuCunZai")%>')
					}
				}).catch(function(error){})		    	
		    },
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
		    //删除
		    delCollectFile(row){
		    	var vm = this , 
					url='${ctx}/cell/collect/doClearImmediateCollectLogFile.action' ,
					id = row.task_id,
					fileName = ''
					params= {
						taskIds:id,
						fileName : fileName,
						isGnb: 1
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
						vm.$refs.ctableLog.refresh()
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

	    	// 点击页面其他地方关闭菜单
	        handerClose(){
	            this.$refs.menus.hide();
	        },	
	    },
	    mounted(){
	    	//this.init();
	    }
	});
	
</script> 	
