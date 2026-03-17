<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#addEnbRebootTask .el-icon-time{
	line-height:1;
}
#addEnbRebootTask .timeItem .el-input__inner{
	width:220px;
}
#addEnbRebootTask .ml45{
	margin-left: 45px
}
#addEnbRebootTask .mt20{
	margin-top: 20px
}
#addEnbRebootTask .taskInput{
	width:680px;
	height:28px;
	line-height:28px;	
}
#addEnbRebootTask .el-form-item__label{
	line-height:26px;
	width:140px;
	text-align:left;
}
#addEnbRebootTask .modeItem .el-radio{
	display:inline-block;
	margin-left:0px;
} 
#addEnbRebootTask .modeItem{
	margin-top:30px;
}
#addEnbRebootTask .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
#addEnbRebootTask .titleStyML{
	margin-left: 20px;
}
#addEnbRebootTask .editButton{
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
	right:130px;
	top:-6px;
}
#addEnbRebootTask .editButton i{
	font-size:14px !important;
}
#addEnbRebootTask .editButton span{
	font-size:12px;
}
.enbDialogStyle .el-icon-circle-info:before{
	color:#CFCFCF;
}
#addEnbRebootTask .el-pairgrid-title{
	top:-6px;
}
</style>

<!-- 新建eNB重启任务 -->
<div id="addEnbRebootTask" style="margin:20px;">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm">
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' prop='taskname' class="ml45 mt20" label-width="120px">
			<el-input maxlength=50 :disabled='showName' v-model="ruleForm.taskname" size="mini" class="taskInput"></el-input>
		</el-form-item>
		<!-- 选择重启类型 -->
		<el-form-item prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" style='display:inline-block;width:380px;margin:0px 0px 20px 45px;' label-width="120px">
			<el-select v-model='ruleForm.product'>	
				<el-option v-for='item in productType' :label="item.name" :value="item.value"></el-option>
			</el-select>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<el-pairgrid :id="'enb_select_list'" v-if='showPairGrid' ref="cpairgrid" @selection-change='selectChange'  :right-url="rightUrl" :left-url="leftUrl" :height="height" row-key="small_cell_code" :query-params="queryParams" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}" style='margin:10px 45px 0px 45px;'>
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
			<template slot='right'>
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
				<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
			</template>
		</el-pairgrid>
		<div v-else style='margin:0px 45px;'>
			<div style='font-size:16px;padding-top:10px;'><%=rb.getString("YiXuanZeJiZhan")%></div>
			<el-ctable :id="'selected_device_list'" ref="stable"  :url="rightUrl" :height="height" pagination="true" :query-params="fileParams">
				<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number"></el-table-column>
				<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name"></el-table-column>
			</el-ctable>
		</div>
		<el-form-item prop='cellCodes' style="margin-left:45px;margin-bottom:30px;">
			<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<el-form-item style='display:inline-block;margin-left:45px;margin-bottom:0px;' prop='status' class='modeItem'>
			<el-radio-group v-model="ruleForm.status" :disabled='showType' @change="statusChange">
				<el-radio label="active" style='margin-right:80px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
				<el-radio label="suspend" style='margin-right:80px;'><%=rb.getString("GuaQi")%></el-radio>
				<el-radio label="timing" style='margin-bottom:0px;margin-right:10px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
			</el-radio-group>
		</el-form-item>
		<el-form-item prop='exetime' style='display:inline-block;margin-bottom:0px;margin-left:15px' class='timeItem'>
			<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="setTimeEnable" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
		</el-form-item>
	</el-form>
	<el-dialog class='enbDialogStyle' :title='dialogTitle' width='630px' :visible.sync='listVisible' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
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
new Vue({
	el:'#addEnbRebootTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else{
				axios.post('${ctx}/task/reboot/taskNameExist.action',stringify({
					taskName:vm.ruleForm.taskname.trim(),
					taskType:'1'
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
			var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
				list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
					return item.length > 0;
				});
				
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
			deviceTitle:['<%=rb.getString("JiZhanLieBiao")%>','<%=rb.getString("YiXuan")%>'],
			setTimeEnable:true,
			rowData : [],
			selection:'',
			ruleForm:{
				taskname:'${addTaskName}',
				cellCodes:'',
				status:'active',
				exetime:'',
				product:''
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
				serial_nubmer:'',
				host_name:'',
				search_text:'',
				group_id:'',
				productValue:''
			},
			queryForm:{
				group_id:'',
			},
			taskId:'',
			defaultTaskName:'',
			showPairGrid:true,
			showName:false,
			showType:false,
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			productType:[],
			addListForm:{
				serialNumber:''
			},
			addListRules:{
				serialNumber:[
					{validator:validatorNum,trigger:'change'}
				]
			},
			listVisible:false,
			dialogTitle:"<%=rb.getString("TianJia")%>"
		}
	},
	methods:{ 
		init:function(){
			var vm = this;
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : vm.queryParams.productValue,
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
			axios.post('${ctx}/cell/version/getProductType.action').then(function(response){
                let data = response.data ? response.data : [];
                let productTypeList = [];
                data.map((item)=>{
                    if(item.value.includes("CR-B4860")){
                        if(item.value == 'CR-B4860/BU'){
                            productTypeList.push({
                                name: item.name.substr(0,8),
                                device_type: item.device_type,
                                value: item.value.substr(0,8),
                            });
                        }
                    }else{
                        productTypeList.push(item);
                    }
                })
				vm.productType = productTypeList;
				vm.ruleForm.product = vm.productType[0].value;
				initForm(vm.$refs.ruleForm);
			}).catch(function(error){
				
			})
		},
		/**
		 * 模糊查询
		 * @param val:查询参数
		*/
		query(val){
			this.queryParams.search_text = val;
			this.$refs.cpairgrid.reload();
		},
		advanceQuery(){ // 高级搜索查询
			var vm = this;
			Object.assign(vm.queryParams,vm.queryForm);
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
			this.selection = selection
		},
		rowClickUpgrade(row){
	        this.rowData = row;
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
	    	var vm = this;
	    	var message = '';
            // 防止多次提交
            if(vmReboot.slideSubmitLoading)return

	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {};
	    			params.timeZone = timeZone;
	    			params.cellCodes = vm.ruleForm.cellCodes;
	    			params.taskName = vm.ruleForm.taskname;
	    			params.status = vm.ruleForm.status;
	    			if(vm.ruleForm.status == 'timing'){
	    				params.time = vm.ruleForm.exetime;
	    			}
	    			params.productType = vm.ruleForm.product;
	    	    	url = '${ctx}/task/reboot/addTask.action'
	    	    	message = '<%=rb.getString("ChengGong")%>'
	    	    	vmReboot.slideSubmitLoading = true;
	    			axios.post(url,stringify(params)).then(function(response){
	    				var data = response.data;
	    				if(data["success"]){
	    					vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            eventBus.$emit('cancel-reboot');
	    				}else{
	    					vm.$message.error(data["message"]);
                            vmReboot.slideSubmitLoading = false;
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
					eventBus.$emit('hide-reboot')
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('hide-reboot')
			}
		},
		setTime(){
			this.ruleForm.exetime = formatDate(new Date(gloableTime));
			this.$refs.ruleForm.validateField('exetime');
		},
		addBatchSn(){
			this.listVisible = true;
		},
		closeBatchSn(){
			this.listVisible = false;
			this.$refs.addListForm.resetFields();
		},
		saveBatchSn(){
			var vm = this, snStr = vm.addListForm.serialNumber || '',
			list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
			
			var params = {
					serialNumbers : list.join(";"),
					productType : vm.ruleForm.product
			}
			vm.$refs.addListForm.validate((valid) => {
				if(valid){
					axios.post("${ctx}/cell/cpeinfos/getTaskCheckSNList.action",stringify(params)).then((res)=>{
						var data = res.data;
						if(data && data.length > 0){
							vm.$refs.cpairgrid.appendCheckedRows(data);
							vm.closeBatchSn();
						}else{
							vm.$message("<%=rb.getString("MeiYouKePiPeiSheBei")%>")
						}
					})
				}
			})
		}
	},
	watch:{
		selection(){
			var data = this.$refs.cpairgrid.getData();
			var cellCodes = '';
			if(data.length != 0){
				data.map(function(item){
					cellCodes += item.small_cell_code + ","
				})
			}
			this.ruleForm.cellCodes = cellCodes
		},
		"ruleForm.status":function(newVal){
			if(newVal == 'timing'){
				this.setTimeEnable = false
			}else{
				this.setTimeEnable = true
			}
			this.$refs.ruleForm.validateField('exetime')
		},
		"ruleForm.rebootType":function(newVal){
			var vm = this;
			var reg = new RegExp('\\\\',"g");
			
			if(newVal == 'reboot_system'){
				vm.queryParams.productValue = "";
			}else if(newVal == 'reboot_stk' || newVal == 'reboot_ru'){
				if(vm.queryParams.productValue != ''){
					vm.queryParams.productValue = "FAP/w+CRw+/CR";
					vm.$refs.cpairgrid.reload()
				}else{
					vm.queryParams.productValue = "FAP/w+CRw+/CR";
				}
			}
			vm.$refs.cpairgrid.clear();
			vm.leftUrl = '${ctx}/task/upgrade/queryCellInfos.action';
			vm.$refs.cpairgrid.reload();
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : vm.queryParams.productValue,
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
		},
		"ruleForm.product":function(newVal){
			var vm = this;
			
			var reg = new RegExp('\\\\',"g");
			vm.$refs.cpairgrid.clear();
			vm.queryParams.productValue = newVal.replace(reg,'');
			vm.leftUrl = '${ctx}/task/upgrade/queryCellInfos.action'
			vm.$refs.cpairgrid.reload();
			
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : newVal.replace(reg,''),
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
		},
	},
	mounted(){
		this.init();
		eventBus.$off('hander-ok').$on('hander-ok',this.submit);
		eventBus.$off('hander-cancel').$on('hander-cancel',this.cancel);
	}
})
</script>