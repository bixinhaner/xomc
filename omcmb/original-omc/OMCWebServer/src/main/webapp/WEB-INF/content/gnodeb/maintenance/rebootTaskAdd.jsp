<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#addGnbRebootTask .el-icon-time{
	line-height:1;
}
#addGnbRebootTask .timeItem .el-input__inner{
	width:220px;
}
#addGnbRebootTask .ml45{
	margin-left: 45px
}
#addGnbRebootTask .mt20{
	margin-top: 20px
}
#addGnbRebootTask .taskInput{
	width:680px;
	height:28px;
	line-height:28px;	
}
#addGnbRebootTask .el-form-item__label{
	line-height:26px;
	width:140px;
	text-align:left;
}
#addGnbRebootTask .modeItem .el-radio{
	display:inline-block;
	margin-left:0px;
} 
#addGnbRebootTask .modeItem{
	margin-top:30px;
}
#addGnbRebootTask .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
#addGnbRebootTask .titleStyML{
	margin-left: 20px;
}
#addGnbRebootTask .editButton{
	padding:0 10px;
	height:24px;
	background:#F2F9FF;
	border-radius:2px;
	line-height:24px;
	cursor:pointer;
	margin-left:10px;
	border:1px solid #1DA3FC;
	display:inline-block;
	position:absolute;
	right:140px;
	top:-6px;
}
#addGnbRebootTask .editButton i{
	font-size:14px !important;
}
#addGnbRebootTask .editButton span{
	font-size:12px;
}
.dialogStyle .el-icon-circle-info:before{
	color:#CFCFCF;
}
#addGnbRebootTask .el-pairgrid-title{
	top:-6px;
}
#addGnbRebootTask .deviceItem{
	display:inline-block;
	margin-right:60px;
}
#addGnbRebootTask .pairgrid-right .el-ctable-toolbar{
	padding: 10px!important;
}
</style>

<!-- 新建gNB重启任务 -->
<div id="addGnbRebootTask" style="margin-top:20px;">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm">
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' prop='taskname' class="ml45 mt20" label-width="120px">
			<el-input maxlength=50  v-model="ruleForm.taskname" size="mini" class="taskInput"></el-input>
		</el-form-item>
		<!-- 选择重启类型 -->
		<el-form-item prop="productValue" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" style='display:inline-block;width:380px;margin:0px 0px 20px 45px;' label-width="120px">
			<el-select v-model='ruleForm.productValue'>	
				<el-option v-for='item in productDataList' :label="item.name" :value="item.value"></el-option>
			</el-select>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<el-pairgrid :id="'enb_select_list'" ref="gnbDevicePairgrid" @selection-change='selectChange'  :right-url="rightUrl" :left-url="leftUrl" :height="height" row-key="small_cell_code" :query-params="queryParams" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}" style='margin:10px 45px 0px 45px;'>
			<template slot="left">
				<el-table-column type="selection" width="45"></el-table-column>
				<el-table-column prop="connection_status" width="50">
					<template slot-scope="scope">
						<div :class="{
							'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
							'':scope.row.have_connected==2,
							'conn_exc':scope.row.connection_status=='Exception',
							'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
					</template>
				</el-table-column>
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
				<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
				<el-table-column prop='product_type' label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>'></el-table-column>
				<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>'></el-table-column>
			</template>
			<template slot='toolbar'>
				<!-- 新建任务 基站列表的高级选择 -->
				<el-form :model='queryForm' ref="queryForm" label-position="top">
					<div style="display: flex;align-items: center;">
						<el-query type="normal" @query="query"  :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'"></el-query>
						<el-popfilter style="margin: 0 5px;"
							type="single"
							label='<%=rb.getString("SheBeiZu")%>'
							v-model="queryForm.group_id"
							:list="groupOptions.map(item=>{return {label:item.text,value:item.value}})"
							@check-change="advanceQuery">
						</el-popfilter>
						<div class="pop-filter-clear" style="margin: 0 5px;" 
							@click="resetQuery">
							<%=rb.getString("QingKongShaiXuan")%>
						</div>
					</div>
				</el-form>
				<div class="editButton" size="mini" @click="addBatchSn">
					<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
					<span><%=rb.getString("PiLiangShuRu")%></span>
				</div>
			</template>
			<template slot='right' class="rightTableCls">
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
				<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
			</template>
		</el-pairgrid>
		<el-form-item prop='cellCodes' style="margin-left:45px;margin-bottom:30px;">
			<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<el-form-item style='display:inline-block;margin-left:45px;margin-bottom:20px;' prop='status' class='modeItem'>
			<el-radio-group v-model="ruleForm.status"  @change="statusChange">
				<el-radio label="active" style='margin-right:80px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
				<el-radio label="suspend" style='margin-right:80px;'><%=rb.getString("GuaQi")%></el-radio>
				<el-radio label="timing" style='margin-bottom:0px;margin-right:10px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
			</el-radio-group>
		</el-form-item>
		<el-form-item prop='exetime' style='display:inline-block;margin-bottom:20px;margin-left:15px' class='timeItem'>
			<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="setTimeEnable" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
		</el-form-item>
	</el-form>
	<el-dialog class='dialogStyle' :title='dialogTitle' width='630px' :visible.sync='listVisible' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
		<el-form ref='addListForm' :rules='addListRules' :model='addListForm' label-position="top">
			<div>
				<label><%=rb.getString("XiaoZhanBianMa")%></label>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='addListForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='saveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeBatchSn'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
</div>

<script type="text/javascript">
var addGnbRebootTask = new Vue({
	el:'#addGnbRebootTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else{
				axios.post('${ctx}/gnb/reboot/taskNameExist.action',stringify({
					taskName:vm.ruleForm.taskname.trim(),
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						if(data["message"] == "true"){
							callback(new Error('<%=rb.getString("RenWuMingChengYiCunZai")%>'))
						}else{
							callback();
						}
					}
				}).catch(function(error){
					callback()
				})
			}
		};
		var validateTime = (rule,value,callback) => {
			if(this.ruleForm.status !== 'timing'){
				callback()
			}else{
				if(value == '' || value==null){
					callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
				}else{
					callback();
				}
			}
		}
		var validatorNum = (rule,value,callback) => {
			var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,45}$/,
				list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});

		    if (serialNumber == null || serialNumber.length == 0) {
				callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
			}else {
				var nameFlag = list.every(function(item,index){
					return temp.test(item)
				})
				if(nameFlag){
					callback()
				}else{
					callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
				}
				
			}
		};
		return {
			leftUrl:'',
			rightUrl:'',
			height:'370px',
			groupOptions:[],
			deviceTitle:['<%=rb.getString("SheBeiLieBiao")%>','<%=rb.getString("YiXuan")%>'],
			setTimeEnable:true,
			selection:'',
			productValue:'',
			ruleForm:{
				taskname:'${addTaskName}',
				cellCodes:'',
				status:'active',
				exetime:'',
				productValue:''
			},
			rules:{
				taskname:[
					{validator:validateName,trigger:'blur'}
				],
				cellCodes:[
					{required:true,message:'<%=rb.getString("QingXuanZeSheBei")%>',trigger:'change'}
				],
				exetime:[
					{type:'date',validator:validateTime,trigger:'change'}
				]
			},
			queryParams:{
				timeZone:timeZone,
				isShowSlave:false,
				isShowRTDDC:'true', // 标识支持RTD 升级 重启
				serial_number:'',
				host_name:'',
				isGnb:1,
				search_text:'',
				group_id:'',
				productValue:''
			},
			queryForm:{
				group_id:'',
			},
			defaultTaskName:'',
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			productDataList:[],
			addListForm:{
				serialNumber:''
			},
			addListRules:{
				serialNumber:[
					{validator:validatorNum,trigger:'change'}
				]
			},
			listVisible:false,
			dialogTitle:'<%=rb.getString("TianJia")%>'
		}
	},
	methods:{ 
		init(){
			var vm = this;
			axios.post("${ctx}/task/upgrade/getProductType.action?isGnb=1").then(function(res){
            	var data = res.data;
		   		
				if(data.length > 0){
					vm.productDataList = data;
					//数组第一条数据为默认的产品类型
					vm.$nextTick(function(){
						vm.leftUrl= '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
    				});
					vm.ruleForm.productValue = vm.productDataList[0].value;
				}else{
					vm.$nextTick(function(){
						vm.leftUrl= '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
    				});
					vm.ruleForm.productValue = '';
				}
			});
			vm.$nextTick(function(){
				initForm(vm.$refs.ruleForm);
				//vm.leftUrl= '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
			})
		},
		/**
		 * 模糊查询
		 * @param val:查询参数
		*/
		query(val){
			var vm = this;
			this.queryParams.search_text = val;
		},
		advanceQuery(){ // 高级搜索查询
			var vm = this;
			Object.assign(vm.queryParams,vm.queryForm);
			vm.$refs.gnbDevicePairgrid.reload();
		},
		resetQuery(){ // 普通搜索查询按钮
			var vm = this,
				params = {
					group_id:'',
				};
			Object.assign(vm.queryForm,params);
			Object.assign(vm.queryParams,params);
		},
		/**
		 * 全选操作
		 * @param selection:选择的数据
		*/
		selectChange(selection){
			var vm = this;
			vm.selection = selection
		},
		// 执行方式改变事件
		statusChange(val){
			var vm = this;
			if(val !== 'timing'){
				vm.ruleForm.exetime = '';
				vm.$refs.ruleForm.clearValidate('exetime')
			}
		},
		submit(){ // 确定按钮 
	    	var vm = this,
				url = '${ctx}/gnb/reboot/saveRebootTask.action',
				message = '',
				params = {
					timeZone:timeZone,
					cellCodes:vm.ruleForm.cellCodes,
					taskName:vm.ruleForm.taskname,
					status:vm.ruleForm.status,
					productValue:vm.ruleForm.productValue
				};
            // 防止多次提交
            if(gNBRebootVue.slideSubmitLoading)return
            
	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			
	    			if(vm.ruleForm.status == 'timing'){
	    				params.time = vm.ruleForm.exetime;
	    			}
                    gNBRebootVue.slideSubmitLoading = true;
	    			axios.post(url,stringify(params)).then(function(response){
	    				var data = response.data;
	    				if(data["success"]){
	    					vm.$message({
	    						message:'<%=rb.getString("ChengGong")%>',
	    						type:'success',
	    					})
							eventBus.$emit('hide-reboot-task');
	    				}else{
	    					vm.$message.error(data["message"]);
                            gNBRebootVue.slideSubmitLoading = false;
	    				}
	    			})
	    		}else{
	    			return false;
	    		}
	    	})
		},
		cancel(){ // 关闭弹窗
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
			if(isFormChanged(this.$refs.ruleForm)){
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					eventBus.$emit('hide-reboot-task');
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('hide-reboot-task');
			}
		},
		setTime(){
			var vm = this;
			vm.ruleForm.exetime = formatDate(new Date(gloableTime));
			vm.$refs.ruleForm.validateField('exetime');
		},
		addBatchSn(){
			var vm = this;
			vm.listVisible = true;
		},
		closeBatchSn(){
			var vm = this;
			vm.listVisible = false;
			vm.$refs.addListForm.resetFields();
		},
		saveBatchSn(){
			var vm = this, snStr = vm.addListForm.serialNumber || '';
				list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
			
			var params = {
					serialNumbers : list.join(";"),
					productType : vm.ruleForm.productValue
			}
			vm.$refs.addListForm.validate((valid) => {
				if(valid){
					axios.post("${ctx}/cell/cpeinfos/getTaskCheckSNList.action",stringify(params)).then((res)=>{
						var data = res.data;
						if(data && data.length > 0){
							vm.$refs.gnbDevicePairgrid.appendCheckedRows(data);
							vm.closeBatchSn();
						}else{
							vm.$message('<%=rb.getString("MeiYouKePiPeiSheBei")%>')
						}
					})
				}
			})
		}
	},
	watch:{
		selection(){
			var vm = this,
				data = this.$refs.gnbDevicePairgrid.getData();
				cellCodes = '',cellCodeList = [];
			if(data.length != 0){
				data.map(function(item){
					cellCodeList.push(item.small_cell_code);
				})
			}
			cellCodes = cellCodeList.join(',');
			vm.ruleForm.cellCodes = cellCodes;
		},
		"ruleForm.status":function(newVal){
			if(newVal == 'timing'){
				this.setTimeEnable = false
			}else{
				this.setTimeEnable = true
			}
			this.$refs.ruleForm.validateField('exetime')
		},
		"ruleForm.productValue":function(newVal){
			var vm = this;
			
			vm.queryParams.productValue = newVal;
			vm.$refs.gnbDevicePairgrid.clear();
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				isGnb : 1,
				productValue : newVal,
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
		},
	},
	mounted(){
		this.init();
		eventBus.$off('save-reboot-task').$on('save-reboot-task',this.submit);
		eventBus.$off('cancel-reboot-task').$on('cancel-reboot-task',this.cancel);
	}
})
</script>