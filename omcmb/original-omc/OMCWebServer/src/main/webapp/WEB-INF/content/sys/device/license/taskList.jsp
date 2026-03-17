<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.queryInfo{
		display:inline-block;
		margin-right:80px;
	}
	.queryInfo label{
		display:block;
	}
	.licenseTask1588SlideCls .el-card__header{
		background:#FBFBFB;
		border-bottom:1px solid #EEEEEE;
	}
	.el-card__body{
		padding:0px;
	}
    .licenseTask1588SlideCls .el-icon-close{
        right: unset !important;
        top: unset !important;
    }
    .licenseTask1588SlideCls .el-card__header .clearfix{
        position: relative;
        margin-right: 40px;
    }
</style>
<div id='licenseTaskList'>
	<el-ctable ref="ctable" time=6 :id="tableId" :url='url' :width='width' :height="height"  :query-params="params" pagination="true">
		<template slot="toolbar" style="position: relative;">
            <div style="display: flex;align-items: center;">
                <span style="font-size:14px;;margin: 0px 10px 0px 20px;font-weight:bold;"><%=rb.getString("RenWuLieBiao")%></span>
                <el-query style="width: 80%;" @query="query" @advance-query="advanceQuery" @reset="resetQuery" placeholder="Task Name" ok-text="<%=rb.getString("ChaXun")%>" reset-text="<%=rb.getString("ChaXunChongZhi")%>">
                    <template slot="form">
                        <div class='queryInfo'>
                            <label><%=rb.getString("RenWuMingCheng")%></label>
                            <el-input v-model='params.task_name' size="mini" style='width:200px'></el-input>
                        </div>
                        <div class='queryInfo'>
                            <label><%=rb.getString("ZhuangTai")%></label>
                            <el-select v-model='params.task_status'>
                                <el-option v-for='item in statusOptions' :label='item.label' :value='item.value'></el-option>
                            </el-select>
                        </div>
                        <div class='queryInfo'>
                            <label><%=rb.getString("JieGuo")%></label>
                            <el-select v-model='params.task_result'>
                                <el-option v-for='item in resultOptions' :label='item.label' :value='item.value'></el-option>
                            </el-select>
                        </div>
                        <div class='queryInfo'>
                            <label><%=rb.getString("KaiShiShiJian")%></label>
                            <el-date-picker v-model='date' size="mini"  value-format="yyyy-MM-dd HH:mm:ss" type="datetimerange" range-separator="——"  start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
                        </div>
                    </template>
                </el-query>
            </div>
            <div class="newIconBoxCls-bt" @click="close1588LicenseTaskListSlide" style="right:20px;top:10px;" tip="<%=rb.getString("GuanBi")%>">
                <span class='el-icon el-icon-close' style="right: 5px;"></span>
            </div>
		</template>
		<el-table-column label='' width="30">
			<template slot-scope="scope">
           			<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
         		</template>
		</el-table-column>
		<el-table-column prop='task_name' label='<%=rb.getString("RenWuMingCheng")%>' width='300'></el-table-column>
		<el-table-column prop='creator' label='<%=rb.getString("ChuangJianZhe")%>' width='80' show-overflow-tooltip="true"></el-table-column>
		<el-table-column prop='start_time' label='<%=rb.getString("KaiShiShiJian")%>'></el-table-column>
		<el-table-column prop='end_time' label='<%=rb.getString("JieShuShiJian")%>'></el-table-column>
		<el-table-column prop='task_status' label='<%=rb.getString("ZhuangTai")%>'>
			<template slot-scope="scope">
				<div v-if="scope.row.task_status == 'Waiting'">
					<span class='el-icon el-icon-status-waiting1'></span><%=rb.getString("DengDai")%>
				</div>
				<div v-if="scope.row.task_status == 'In Progress'">
					<span class='el-icon el-icon-status-inProgress'></span><%=rb.getString("JinXingZhong")%>
				</div>
				<div v-if="scope.row.task_status == 'Suspend'">
					<span class='el-icon el-icon-status-suspend'></span><%=rb.getString("ZanTing")%>
				</div>
				<div v-if="scope.row.task_status == 'End'">
					<span class='el-icon el-icon-status-terminate'></span><%=rb.getString("YiJieShu")%>
				</div>
				<div v-if="scope.row.task_status == 'Termination'">
					<span class='el-icon el-icon-status-terminate'></span><%=rb.getString("ZhongZhi")%>
				</div>
			</template>
		</el-table-column>
		<el-table-column prop='task_progress' label='<%=rb.getString("JinDu")%>'></el-table-column>
		<el-table-column prop='task_result' label='<%=rb.getString("JieGuo")%>' :formatter='resultFmt'></el-table-column>
	</el-ctable>
	<el-cmenu ref="menu" :data="menus" @click="clickMenu"></el-cmenu>
	<el-slide class='licenseTask1588SlideCls' ref="listSlide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition"
	 :height="slideHeight" :modal='slideModal'  :width="slideWidth" @ok='saveEditTask' @cancel='cancelEditSlide' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	 </el-slide>
</div>
<script>
	var licenseTask1588Vue = new Vue({
		el:'#licenseTaskList',
		data(){
			return{
				tableId:'licTaskListTable',
				url:'${ctx}/cell/1588License/getTask.action',
				width:'100%',
				height:'100%',
				params:{
					time_zone:timeZone,
					task_name:'',
					task_status:'',
					task_result:'',
					start_time:'',
					end_time:'',
					search_text:''
				},
				date:[],
				statusOptions:[
					{
						label:'<%=rb.getString("QuanBu")%>',
						value:''
					},
					{
						label:'<%=rb.getString("DengDai")%>',
						value:'Waiting'
					},
					{
						label:'<%=rb.getString("JinXingZhong")%>',
						value:'In Progress'
					},
					{
						label:'<%=rb.getString("ZanTing")%>',
						value:'Suspend'
					},
					{
						label:'<%=rb.getString("YiJieShu")%>',
						value:'End'
					}
				],
				resultOptions:[
					{
						label:'<%=rb.getString("QuanBu")%>',
						value:''
					},
					{
						label:'<%=rb.getString("ChengGong")%>',
						value:'Success'
					},
					{
						label:'<%=rb.getString("BuFenChengGong")%>',
						value:'Partial Success'
					},
					{
						label:'<%=rb.getString("ShiBai")%>',
						value:'Fail'
					},
					{
						label:'<%=rb.getString("WeiZhiXing")%>',
						value:'Unexecuted'
					}
				],
				menus:[],
				rowData:[],
				slideUrl:'',
				slideTitle:'',
				slideHeader:false,
				slideFooter:false,
				slidePosition:'top',
				slideModal:false,
				slideWidth:'',
				slideHeight:'',
				exportFlag:false,
				oper_type:''
			}
		},
		methods:{
			handerClose(){
				this.$refs.menu.hide();
			},
			query(val){
				this.resetQuery();
    			this.params.search_text  = val;
    			this.$refs.ctable.refresh();
			},
			advanceQuery(){
				this.params.search_text = "";
    			if(this.date != null){
    				this.params.start_time = this.date[0];
	    			this.params.end_time = this.date[1];
    			}
    			this.$refs.ctable.refresh()
			},
			resetQuery(){
				this.params.task_name = '';
				this.params.task_status = '';
				this.params.task_result = '';
    			this.date = []
			},
			optClick(row,ev){//软件升级点击操作出现下拉菜单  1.等待   2.进行中  3.暂停  4.已结束 5.终止中 6.暂停中
				var vm = this;
				vm.rowData = row;
				var status = row.task_status;
    			var startShow = false,
    				startFlag = false,
    				waitFlag = false,
    				endFlag = false,
    				delFlag = false,
    				editFlag = false,
    			 	awaitShow = false
		    	if(status == 'Waiting'){//等待
		    		startShow = true;
		    		endFlag = true;
		    	}
		    	if(status == 'In Progress'){//进行中
		    		awaitShow = true;
		    		delFlag = true;
		    		editFlag = true;
		    	}
		    	if(status == 'Suspend'){//挂起
		    		startShow = true;
		    		//delFlag = true;
		    		//editFlag = true;
		    	}
		    	if(status == 'End' || status == 'Termaination'){//已结束/终止中
		    		startShow = true;
		    		startFlag = true;
		    		endFlag = true;
		    		editFlag = true;
		    	}
				vm.menus= [
					  {label:'<%=rb.getString("JieGuo")%>',cls:"el-icon el-icon-operation-result",code:'result'},
					  {label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start",code:'start',show:startShow},
					  {label:'<%=rb.getString("ZanTing")%>',cls:"el-icon el-icon-operation-awaiting",code:'stop',show:awaitShow},
			          {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:"el-icon el-icon-operation-terminate",code:'terminate',disable:endFlag},
			          {label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
			          {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:'edit',disable:editFlag},
			          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del',disable:delFlag}
			    ]
		    	vm.$nextTick(function(){
		    		document.body.click();
			    	vm.$refs.menu.show(ev);
		    	});
			},
			clickMenu(ev){
				var codes = {
						result:this.viewTaskResult,
						start:this.activeTask,
						stop:this.suspendTask,
						terminate:this.terminateTask,
						del:this.deleteTask,
	    	    		edit:this.editLicenseTask,
	    	    		info:this.viewLicenseTask
	    	    	}
	    	    	if(codes[ev.code]){
	    	    		codes[ev.code]()
	    	    	}
			},
			viewTaskResult(){
				var vm = this;
				vm.oper_type = 'viewTask'
	        	vm.slideHeader = false
	        	vm.slideTitle = 'Task Result'
	    	    vm.slideUrl = '${ctx}/cell/1588License/goTaskProgressPage.action?task_id='+vm.rowData.task_id
	    	    vm.slideFooter = false
	    	    vm.slidePosition = 'bottom'
	    	    vm.slideHeight = '350px'
	    	    vm.slideWidth = '100%'
	    	    vm.$refs.listSlide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    	eventBus.$emit('view-result',vm.rowData.task_id)
	    	    });
			},
			activeTask(){
				var vm = this;
    	    	axios.post('${ctx}/cell/1588License/activeTask.action',stringify({
    	    		task_id : vm.rowData.task_id,
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			vm.$refs.ctable.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
			},
			suspendTask(){
				var vm = this;
    	    	axios.post('${ctx}/cell/1588License/suspendTask.action',stringify({
    	    		task_id : vm.rowData.task_id
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			vm.$refs.ctable.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
			},
			terminateTask(){
				var vm = this;
    	    	axios.post('${ctx}/cell/1588License/terminateTask.action',stringify({
    	    		task_id : vm.rowData.task_id
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			vm.$refs.ctable.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
			},
			deleteTask(){
				var vm = this;
    	    	vm.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
    	    		customClass:'warningConfirm',
    	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
    	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
    	    		type:'warning',
    	    		closeOnClickModal:false
    	    	}).then(() => {
    	    		axios.post('${ctx}/cell/1588License/clearTask.action',stringify({
    		    		task_id:vm.rowData.task_id
    		    	})).then(function(response){
    		    		var data = response.data;
    		    		if(data["success"]){
    		    			vm.$refs.ctable.refresh()
    		    			vm.$message({
    			    			type:'success',
    			    			message:'<%=rb.getString("ShanChuChengGong")%>'
    			    		})
    		    		}else{
    		    			vm.$message.error(data["message"])
    		    		}
    		    	}).catch(function(error){
    		    		
    		    	})
    	    	}).catch()
			},
			editLicenseTask(){
				var vm = this;
				vm.oper_type = 'edit'
	        	vm.slideHeader = true
	        	vm.slideTitle = '<%=rb.getString("XiuGai")%>'
	    	    vm.slideUrl = '${ctx}/cell/1588License/goAddTaskPage.action'
	    	    vm.slideFooter = true
	    	    vm.slidePosition = 'left'
	    	    vm.slideHeight = '100%'
	    	    vm.slideWidth = '1080px'
	    	    vm.$refs.listSlide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    	eventBus.$emit('get-info',vm.rowData.task_id,'edit')
	    	    });
			},
			viewLicenseTask(){
				var vm = this;
				vm.oper_type = 'view'
	        	vm.slideHeader = true
	        	vm.slideTitle = '<%=rb.getString("ChaKan")%>'
	    	    vm.slideUrl = '${ctx}/cell/1588License/goAddTaskPage.action'
	    	    vm.slideFooter = false
	    	    vm.slidePosition = 'left'
	    	    vm.slideHeight = '100%'
	    	    vm.slideWidth = '1080px'
	    	    vm.$refs.listSlide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    	eventBus.$emit('get-info',vm.rowData.task_id,'view')
	    	    });
			},
			cancelEditSlide(){
				if(this.oper_type == 'edit'){
					eventBus.$emit('cancel-edit')
				}else{
					this.$refs.listSlide.hide();
				}
			},
			closeEditTask(){
				this.$refs.listSlide.hide();
			},
			closeEditTaskSuc(){
				this.$refs.listSlide.hide();
				this.$refs.ctable.refresh();
			},
			saveEditTask(){
				eventBus.$emit('save-edit-task');
			},
			resultFmt(row,column,cellValue,index){
				var resultObj = {
    					"Success" : "<%=rb.getString("ChengGong")%>",
    					"Partial Success" : "<%=rb.getString("BuFenChengGong")%>",
    					"Fail" : "<%=rb.getString("ShiBai")%>",
    					"Unexecuted" : "<%=rb.getString("WeiZhiXing")%>",
    					"" : ""
    			}
    			return resultObj[cellValue];
			},
            close1588LicenseTaskListSlide(){
                eventBus.$emit('hide-task')
            },
		},
		mounted(){
			eventBus.$off('hide-edit').$on('hide-edit',this.closeEditTask);
			eventBus.$off('save-edit-suc').$on('save-edit-suc',this.closeEditTaskSuc)
		}
	})
</script>