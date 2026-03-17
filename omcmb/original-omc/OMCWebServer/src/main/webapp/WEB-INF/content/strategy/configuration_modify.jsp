<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<style>
	#commonNeighborCellConfigPage {
		width: 100%;	
		height: 100%;
		overflow: hidden;
	}
	#commonNeighborCellConfigPage .title-text::after{
		display:none;
	}
	#commonNeighborCellConfigPage .el-form {
		padding: 30px 40px;
	}
	#commonNeighborCellConfigPage .paramPoolWarp {
		padding: 20px 30px;
	}
	#commonNeighborCellConfigPage .paramPoolWarp .el-form-item {
		 width: 49%; 
		 display: inline-block; 
		 flex-direction: row; 
	}
	#commonNeighborCellConfigPage .paramPoolWarp .el-form-item .el-input { width: 200px; }
	#commonNeighborCellConfigPage .validateItem .el-input-group__append { border:none; background:none; padding: 0px 10px; }
	#commonNeighborCellConfigPage .is-error .el-input-group__append { color:#FA5555; }
	#commonNeighborCellConfigPage .validateItem .el-input__inner { width: 200px; }
	#commonNeighborCellConfigPage .validateItem .el-input-group__append{ border:none; background:none; }
	#commonNeighborCellConfigPage .validateItem .el-form-item__error { display:none; }

</style>
<div id='commonNeighborCellConfigPage'>
	<el-form ref="cellConfigForm" :model="cellConfigForm" :rules="cellConfigRules" label-position="top" >
		<div class='group-title'>
			<span class='title-icon'></span>
			<span class='title-text'><%=rb.getString("JiChuPeiZhi")%></span>
		</div>
		<!--gsm 邻区-->
		<div class="paramPoolWarp" v-if="cellActiveName == 'gsmNeighborCellConfig'">
			<el-form-item label='<%=rb.getString("XiaoQuBianHao")%>' prop="cellIndex" class='inputCommon validateItem'>
				<el-select v-model='cellConfigForm.cellIndex'>
					<el-option label='Cell 1' :value='1'></el-option>
					<el-option label="Cell 2" :value='2'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label='ARFCN' prop="arfcn" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.arfcn">
					<template slot="append"><%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~1023</template>
				</el-input>
			</el-form-item>
			<el-form-item label='PLMN' prop="plmn" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.plmn">
					<template slot="append"><%=rb.getString("ZhengXing")%>, Length<%=rb.getString("MaoHao")%> 5~6 Digit</template>
				</el-input>
			</el-form-item>
			<el-form-item label='<%=rb.getString("WeiZhiQuXinXi")%>' prop="lac" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.lac">
					<template slot="append"><%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~65535</template>
				</el-input>
			</el-form-item>
			<el-form-item label='<%=rb.getString("JiZhanShiBieMa")%>' prop="bsic" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.bsic">
					<template slot="append"><%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~63</template>
				</el-input>
			</el-form-item>
			
			<el-form-item label='<%=rb.getString("XIAOQUID")%>' prop="cellId" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.cellId">
					<template slot="append"><%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~65535</template>
				</el-input>
			</el-form-item>
			<el-form-item label='<%=rb.getString("JiHuoZhuangTai")%>' prop="activeState" class='inputCommon validateItem'>
				<el-select v-model='cellConfigForm.activeState'>
					<el-option label='<%=rb.getString("JiHuo")%>' value='true'></el-option>
					<el-option label='<%=rb.getString("QuJiHuo")%>' value='false'></el-option>
				</el-select>
			</el-form-item>
		</div>
		<!--5g 邻区-->
		<div class="paramPoolWarp" v-if="cellActiveName == 'gnbNeighborCellConfig'">
			<el-form-item label='<%=rb.getString("XiaoQuBianHao")%>' prop="cellIndex" class='inputCommon validateItem'>
				<el-select v-model='cellConfigForm.cellIndex'>
					<el-option label='Cell 1' :value='1'></el-option>
					<el-option label="Cell 2" :value='2'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label='PLMN' prop="plmn" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.plmn">
					<template slot="append"><%=rb.getString("ZhengXing")%>, Length<%=rb.getString("MaoHao")%> 5~6 Digit</template>
				</el-input>
			</el-form-item>
			<el-form-item label='SSB' prop="ssb" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.ssb">
					<template slot="append"><%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~3279165</template>
				</el-input>
			</el-form-item>

			<el-form-item label='<%=rb.getString("GNBBiaoShi")%>' prop="gnbId" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.gnbId">
					<template slot="append"><%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~4294967295</template>
				</el-input>
			</el-form-item>
			<el-form-item label='<%=rb.getString("GNBBiaoShiChangDu")%>' prop="gnbIdLength" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.gnbIdLength">
					<template slot="append"><%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：22~32</template>
				</el-input>
			</el-form-item>
			
			<el-form-item label='<%=rb.getString("XIAOQUID")%>' prop="cellId" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.cellId">
					<template slot="append"><%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~16383</template>
				</el-input>
			</el-form-item>
			<el-form-item label='<%=rb.getString("PCI")%>' prop="pci" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.pci">
					<template slot="append"><%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~1007</template>
				</el-input>
			</el-form-item>
			<el-form-item label='QOFFSET' prop="qoffset" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.qoffset">
					<template slot="append"><%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：-24~24</template>
				</el-input>
			</el-form-item>
			<el-form-item label='TAC' prop="tac" class='inputCommon validateItem'>
				<el-input v-model.trim="cellConfigForm.tac">
					<template slot="append"><%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~16777215</template>
				</el-input>
			</el-form-item>
		</div>
	</el-form>
</div>

<script>
var commonNeighborCellConfigVue = new Vue({
	el:'#commonNeighborCellConfigPage',
	data(){
		var vm =this,
			validateRange = (rule, value, callback) => {
				var reg = /^[0-9]*$/,
					message = rule.message;
					min = rule.min,
					max = rule.max;

				if(value === ''){
					callback(new Error(message))
				}else{
					if(reg.test(value)){
						if(value >= min && value <= max){
							callback();
						}else{
							callback(new Error(message))
						}
					}else{
						callback(new Error(message))
					}
				}
			},
			validatePlmn = (rule, value, callback) => {
				var reg = /^[0-9]{5,6}$/;

				if(value === ''){
					callback(new Error('<%=rb.getString("ZhengXing")%>, Length<%=rb.getString("MaoHao")%> 5~6 Digit'));
				}else if(!reg.test(value)){
					callback(new Error('<%=rb.getString("ZhengXing")%>, Length<%=rb.getString("MaoHao")%> 5~6 Digit'));
				}else{
					callback();
				}
			},
			validateQOffsetRange = (rule, value, callback) => {
				var reg = /^-?[0-9]+.?[0-9]*$/;

				if(value === ''){
					callback(new Error('<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：-24~24'))
				}else if(!reg.test(value) || (value < -24 || value > 24)){
					callback('<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：-24~24')
				}else{
					callback()
				}
			},
			//5g: 0~16383;  gsm:0-65535
			validateCellId = (rule, value, callback) => {
				var reg = /^[0-9]*$/,
					message,
					min = 0,
					max;
				if(vm.cellActiveName == 'gsmNeighborCellConfig'){
					max = 65535;
					message = '<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~65535';
				}else if(vm.cellActiveName == 'gnbNeighborCellConfig'){
					max = 16383;
					message = '<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~16383';
				}
				if(value === ''){
					callback(new Error(message))
				}else{
					if(reg.test(value)){
						if(value >= min && value <= max){
							callback();
						}else{
							callback(new Error(message))
						}
					}else{
						callback(new Error(message))
					}
				}
			}
			
		return {
			cellActiveName:'',
			cellRowData:{},
			//重复参数 plmn, cellId
			cellConfigForm:{
				arfcn:'',
				plmn:'',
				lac:'',
				bsic:'',
				cellId:'',
				activeState:'true',
				cellIndex: '1',
				ssb:'',
				gnbId:'',
				gnbIdLength:'',
				pci:'',
				qoffset:'',
				tac:''
			},
			cellConfigRules:{
				arfcn:[
					{required: true, min: 0, max: 1023, validator: validateRange, message: '<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~1023', trigger: 'blur'}
				],
				plmn:[
					{required: true, validator: validatePlmn, trigger:'blur'},
				],
				//BAIBLQ: 65535; B4860: 65533
				lac:[
					{required: true, min: 0, max: 65535, validator: validateRange ,message: '<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~65535', trigger: 'blur'}
				],
				bsic:[
					{required: true, min:0, max: 63, validator: validateRange, message: '<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~63', trigger: 'blur'}
				],
				//5g: 0~16383;  gsm:0-65535
				cellId:[
					{required: true, validator: validateCellId, trigger: 'blur'}
				],
				
				ssb:[
					{required: true, min:0, max: 3279165, validator: validateRange, message: '<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~3279165', trigger: 'blur'}
				],
				gnbId:[
					{required: true, min:0, max: 4294967295, validator: validateRange, message: '<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~4294967295', trigger: 'blur'}
				],
				gnbIdLength:[
					{required: true, min:22, max: 32, validator: validateRange, message: '<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：22~32', trigger: 'blur'}
				],
				pci:[
					{required: true, min:0, max: 1007, validator: validateRange, message: '<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~1007', trigger: 'blur'}
				],
				qoffset:[
					{required: true, min:-24, max: 24, validator: validateQOffsetRange, message: '<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：-24~24', trigger: 'blur'}
				],
				tac:[
					{required: true, min:0, max: 16777215, validator: validateRange, message: '<%=rb.getString("ZhengXing")%>, <%=rb.getString("FanWei")%>：0~16777215', trigger: 'blur'}
				]
			}
		}
	},
	methods:{
		modifyCellConfig(id, radioTab, rowData){
			var vm = this;

			vm.cellActiveName = radioTab;
			vm.cellRowData = rowData;

			Object.assign(vm.cellConfigForm, rowData);
			/*if(rowData.cellIndex){

			}else{
				vm.cellConfigForm.cellIndex = '1';
			}
			if(rowData.activeState){
				
			}else{
				vm.cellConfigForm.activeState = 'true';
			}*/
			vm.$nextTick(function(){
				initForm(vm.$refs.cellConfigForm);
			})
		},
		cellConfigSubmit(){
			var vm = this, url = '', params = {};

			params.id = vm.cellRowData.id;
			params.cellIndex = vm.cellConfigForm.cellIndex;
			params.plmn = vm.cellConfigForm.plmn;
			params.cellId = vm.cellConfigForm.cellId;
			//GMS领区配置
			if(vm.cellActiveName == 'gsmNeighborCellConfig'){
				url = '${ctx}/task/BatchConfiguration/updateGSMNcell.action';
				params.arfcn = vm.cellConfigForm.arfcn;
				params.lac = vm.cellConfigForm.lac;
				params.bsic = vm.cellConfigForm.bsic;
				params.activeState = vm.cellConfigForm.activeState;
			}else {
				url = '${ctx}/task/BatchConfiguration/updateGNBNcell.action';

				params.ssb = vm.cellConfigForm.ssb;
				params.gnbId = vm.cellConfigForm.gnbId;
				params.gnbIdLength = vm.cellConfigForm.gnbIdLength;
				params.pci = vm.cellConfigForm.pci;
				params.qoffset = vm.cellConfigForm.qoffset;
				params.tac = vm.cellConfigForm.tac;
			}
			
			vm.$refs.cellConfigForm.validate((valid) => {
				if(valid){
					axios.post(url,stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
								message:'<%=rb.getString("XiuGai")%><%=rb.getString("ChengGong")%>',
								type:'success',
							})
                            eventBus.$emit('cancel-slide')
						}else{
							vm.$message.error(data["message"])
						}
					})
				}
			})
		},
		cancelCellConfig(){
			var vm = this;

			if(isFormChanged(vm.$refs.cellConfigForm)){
				vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>',QueRen,{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					eventBus.$emit('cancel-slide')
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('cancel-slide')
			}		
		}
	},
	mounted(){
		eventBus.$off('modify-config').$on('modify-config', this.modifyCellConfig)
		eventBus.$off('save-config').$on('save-config',this.cellConfigSubmit)
		eventBus.$off('cancel-config').$on('cancel-config',this.cancelCellConfig)
	}
})
</script>