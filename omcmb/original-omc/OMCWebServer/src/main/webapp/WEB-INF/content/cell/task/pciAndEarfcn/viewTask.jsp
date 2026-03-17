<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.gridClass .el-input__inner{
		height:26px;
		line-height:26px;
		width:80px;
	}
	.earfcnClass{
		width:auto;
	}
	.el-switch__core{
		height:17px;
	}
	.el-switch__core:after{
		width:13px;
		height:13px;
	}
	.el-switch.is-checked .el-switch__core::after{
		margin-left:-16px;
	}
	.errorBorder{
		border:1px solid red;
	}
	.errorMsg .el-form-item__error{
		left:30px;
	}
	.timeError .el-form-item__error{
		left:30px;
		top:121%;
	}
	.queryInfo{
		margin-right:65px;
	}
</style>
<div id='viewPciTaskEnb' style="padding:20px;">
	<el-form ref='enbPciForm' :model="enbPciForm" label-position="top">
		<div class="group-title not-extend">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='Task Name' style='margin-left:30px;' prop='taskName'>
			<el-input v-model='enbPciForm.taskName' disabled=true maxlength=100 size="mini" style="width:600px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		<div class="group-title not-extend">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiZhanXuanZe")%></span>
		</div>
		<el-ctable style='margin-left:30px;' ref="view_enb" :rownumber="true" id="viewEnb_table" :url="viewEnbUrl" height="300px" page-size=20 pagination="true">
			<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' width="200" prop='serial_number'></el-table-column>
			<el-table-column label='<%=rb.getString("HostName")%>' width="300" prop="host_name"></el-table-column>
			<el-table-column label='<%=rb.getString("PinDian")%>' width="150" prop="earfcn"></el-table-column>
			<el-table-column label='PCI' width="200" prop="pci"></el-table-column>
			<el-table-column label='Binding CPE' prop="cpe_flag">
				<template slot-scope="scope">
					<el-switch  :value=scope.row.cpe_flag==1?true:false active-color='#13ce66' inactive-color='#bbb'></el-switch>
				</template>
			</el-table-column>
		</el-ctable>
		<div class="group-title not-extend">
			<span class="title-icon"></span>
			<span class="title-text">Binding CPEs</span>
		</div>
		<el-ctable style='margin-left:30px;' ref="add_enb_table" :rownumber="true" id="view_bindCpe_table" :url="bindCpeUrl" height="300px" page-size=20 pagination="true">
			<el-table-column label='eNB SN' width="100" prop='enb_serial_number'></el-table-column>
			<el-table-column label='eNB Name' width="200" prop="host_name"></el-table-column>
			<el-table-column label='<%=rb.getString("CPEBianMa")%>' width="150" prop="cpe_serial_number"></el-table-column>
			<el-table-column label='<%=rb.getString("CPEName")%>' width="200" prop="cpe_host_name"></el-table-column>
			<el-table-column label='<%=rb.getString("PinDian")%>' width="200" prop="cpe_earfcn_before"></el-table-column>
			<el-table-column label='PCI' width="100" prop="cpe_pci_before"></el-table-column>
			<el-table-column label='Group' width="200" prop="group_name"></el-table-column>
			<el-table-column label='Earfcn(To)' width='200' prop="cpe_earfcn_after"></el-table-column>
			<el-table-column label='PCI(To)' width='100' prop="cpe_pci_after"></el-table-column>
		</el-ctable>
		<div class="group-title not-extend" style='margin-top:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<div style='height:48px;border:1px solid #DEDFE6;margin-left:30px;display:flex;'>
			<el-form-item>
				<el-radio-group v-model='enbPciForm.status' style='margin-top:17px;' disabled=true>
					<el-radio label='active' style='margin-right:110px;margin-left:20px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
					<el-radio label='suspend' style='margin-right:110px;'><%=rb.getString("GuaQi")%></el-radio>
					<el-radio label='timing'><%=rb.getString("DingShiZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item prop='exetime' class='errorMsg'>
				<el-date-picker v-model="enbPciForm.exetime" style='margin-top:10px;vertical-align:middle;margin-left:25px;' :disabled="setTimeEnable" value-format="yyyy-MM-dd HH:mm:ss" type="datetime" :picker-options="pickerOptions"></el-date-picker>	
			</el-form-item>
		</div>
	</el-form>
</div>

<script>
	var addEnb = new Vue({
		el:'#viewPciTaskEnb',
		data(){
			return{
				enbPciForm:{
					taskName:'${addTaskName}',
					status:'timing',
					exetime:''
				},
				viewEnbUrl:'',
				height:'360px',
				bindCpe:true,
				bindCpeUrl:'',
				pickerOptions:{
					disabledDate(time){
						return time.getTime()< Date.now()-8.64e7;
					}
				},
				setTimeEnable:true,
			}
		},
		methods:{
			//  在更多按钮里 信息  页面 获取数据初始化
			getEnbInfo(){
				var vm = this;
				//获取基本信息
				axios.post('${ctx}/task/pcilock/getEnbPciLockTaskInfo.action?taskId='+"${taskId}"+"&timeZone="+timeZone).then(function(response){
					let data = response.data;
					vm.enbPciForm.taskName = data.taskName;
					vm.enbPciForm.status = data.executeType;
					if(data.quartzTime == null || data.quartzTime == ''){
						vm.enbPciForm.exetime = '';
					}else{
						vm.enbPciForm.exetime = data.quartzTime;
					}
				})
				//获取已经选择的enb设备列表
				vm.viewEnbUrl = '${ctx}/task/pcilock/getSelectedList.action?taskId='+"${taskId}";
				vm.bindCpeUrl = '${ctx}/task/pcilock/getBindingCpeList.action?taskId='+"${taskId}";
			}
		},
		mounted(){
			eventBus.$off('view-enb-info').$on('view-enb-info',this.getEnbInfo);
		}
	})
</script>