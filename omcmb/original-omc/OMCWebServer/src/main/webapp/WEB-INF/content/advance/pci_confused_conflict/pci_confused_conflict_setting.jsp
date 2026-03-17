<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#PCIConfusedSetting .el-form-item{
	margin-bottom: 20px;
}
#PCIConfusedSetting .el-form-item .el-input{
	padding-top: 5px;
}
#PCIConfusedSetting .el-form-item__error{
	white-space: nowrap!important;
	padding-top: 0px!important;
}

#PCIConfusedSetting .titleStyML{
	margin-left: 40px;
}
#PCIConfusedSetting .confusedPageTitle{
	height: 60px;
	background-color: #FFF;
	display: flex;
	align-items: center;
	padding-left: 80px;
}
#PCIConfusedSetting .confusedPageTitle>div{
	margin-right: 80px;
}

#PCIConfusedSetting .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	margin-top: 20px;
	height: 1px;
	margin-bottom: 30px; 
}
#PCIConfusedSlide .slide-content{
	padding: 20px 0 20px 0px!important;
}
</style>
</style>
<!--PCI 冲突混淆过滤  设置-->
<div id="PCIConfusedSetting" style="margin-top:20px;">
	
	<el-form :model='settingForm' :rules="rules" ref="settingForm" label-position="left">
		<el-form-item prop='detectorSwitch'>
			<div class="confusedPageTitle">
				<div style="margin-right:20px;width:120px"><%=rb.getString("JianCeKaiGuan")%></div>
				<el-switch v-model="settingForm.detectorSwitch" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#CFCFCF" @change="detectorSwitchChange"></el-switch>
			</div>
		</el-form-item>
		<div class="confusedPageTitle">
			<el-form-item prop='optimizeSwitch'>
				<div style="display:flex;align-items: center;">
					<div style="margin-right:20px;width:120px"><%=rb.getString("YouHuaKaiGuan")%></div>
					<el-switch v-model="settingForm.optimizeSwitch" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#CFCFCF" :disabled="settingForm.detectorSwitch == '0'" @change="optimizeSwitchChange"></el-switch>
				</div>
			</el-form-item>
			<el-form-item prop='controlledType'>
				<div style="display:flex;align-items: center;">
					<el-radio v-model="settingForm.controlledType" label="1" :disabled="settingForm.optimizeSwitch == '0'"><%=rb.getString("ShouKongMeShi")%></el-radio>
					<el-radio v-model="settingForm.controlledType" label="0" :disabled="settingForm.optimizeSwitch == '0'"><%=rb.getString("ZiYouMeShi")%></el-radio>
				</div>
			</el-form-item>
			
		</div>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<div style='margin:20px 50px 0px 70px;height:300px;'>
			<el-ctable 
				id="confusedDeviceGroup"  
				@selection-change='deviceGroupSelect' border=true 
				ref="confusedDeviceGroupTable" 
				:default-checked="deviceChecked"
				row-key="id" 
				:query-params="queryDeviceForm" :url="deviceGroupUrl" style="height:100%;border:1px solid #E9E9E9;">
				<el-table-column type="selection" reserve-selection width="55" key="enbs"></el-table-column>
				<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
			</el-ctable>
		</div>
		<el-form-item prop='cellCodes' style="margin-left:70px;margin-top:5px;">
			<el-input v-model='settingForm.cellCodes' v-show="false"></el-input>
		</el-form-item>
		<el-form-item  style="margin-left:70px;margin-top:10px;" :label="rangeLabel" prop='rangeStart' class="rangeClass" label-width="150px">
			<el-input v-model.trim='settingForm.rangeStart' oninput="if(value != ''){value= Number(value.replace(/[^\d]/g,''))}" maxlength="4" style="width:60px;"></el-input>
			<span style="margin:0px 5px;">-</span>
			<el-input v-model.trim='settingForm.rangeEnd' oninput="if(value != ''){value= Number(value.replace(/[^\d]/g,''))}" maxlength="4" style="width:60px;"></el-input>
		</el-form-item>
		<el-form-item prop='reuseDistance' style="margin-left:70px;" label="<%=rb.getString("ZuiXiaoFuYongJuLi") %>" :label-width="reuseDistanceLableWidth">
			<el-input v-model.trim='settingForm.reuseDistance' style="width:160px;" oninput="if(value != ''){value= Number(value.replace(/[^\d]/g,''))}" maxlength="4"></el-input>
			<span style="margin-left:5px;"><%=rb.getString("FanWei")%>：1-1000</span>
		</el-form-item>
	</el-form>
</div>

<script type="text/javascript">
new Vue({
	el:'#PCIConfusedSetting',
	data(){
		var vm = this;
		var validateRangeStart = (rule,value,callback) => {
			var rangeEnd = vm.settingForm.rangeEnd,
				start = 0,end = 1007,
				errorText = '<%=rb.getString("PCIFanWei")%>' + '(' + start + '-' + end + ')'
			if(value && rangeEnd){
				if(parseInt(value,10) < start || parseInt(rangeEnd,10) > end){
					callback(new Error(errorText))
				}else if(parseInt(value,10) >= parseInt(rangeEnd,10)){
					callback(new Error(errorText))
				}else{
					callback();
				}
			}else{
				callback(new Error('<%=rb.getString("FanWeiBuNengWeiKong")%>'))
			}
		};
		var validateReuseDistance = (rule,value,callback) => {
			var maxReuseDistance = 1000,
				errorText = '<%=rb.getString("FanWei")%>' + '(1-' + maxReuseDistance  + ')';
			if(value){
				if(value > maxReuseDistance || value<1){
					callback(new Error(errorText))
				}else{
					callback()
				}
			}else{
				callback(new Error('<%=rb.getString("FanWeiBuNengWeiKong")%>'))
			}
		};
		
		return {
			selection:'',
			deviceGroupUrl:'${ctx}/system/deviceGroup/getDeviceGroupList.action',
			settingForm:{
				controlledType:'1',
				detectorSwitch:'1',
				optimizeSwitch:'1',
				cellCodes:'',
				rangeStart:'',
				rangeEnd:'',
				reuseDistance:'',
			},
			rules:{
				cellCodes:[
					{required:true,message:'<%=rb.getString("QingXuanZeSheBeiZu")%>',trigger:'change'}
				],
				rangeStart:[
					{validator:validateRangeStart,trigger:'blur'}
				],
				reuseDistance:[
					{validator:validateReuseDistance,trigger:'blur'}
				],
			},
			queryDeviceForm:{

			},
			deviceChecked:[],
		}
	},
	watch:{
		selection(){
			var data = this.selection;
			let cellCodes = "";
			data.map(function(item){
				cellCodes += item.id+','
			})
			this.settingForm.cellCodes = cellCodes;
		},
	},
	computed:{
		rangeLabel:function(){
			return '<%=rb.getString("PCIFanWei")%>' + '(0-1007)';
		},
		reuseDistanceLableWidth:function(){
			return language == 'en'? '280px' : '150px';
		},
	},
	methods:{ 
		// 初始化
		init(){
			var vm = this;
			axios.post("${ctx}/pci/getSettings.action").then((res) => {
				var data = res.data;
				vm.settingForm.detectorSwitch = data.pciCheckSwitch;
				vm.settingForm.optimizeSwitch = data.pciOptimizeSwitch;
				vm.settingForm.controlledType = data.pciControlSwitch;
				vm.settingForm.reuseDistance = data.minMultiplexingDistance;
				vm.settingForm.cellCodes = data.deviceGroupIds;
				var deviceChecked = data.pciDeviceGroupIds.split(',');
				if(deviceChecked.length !== 0){
					deviceChecked.map((item)=>{
						vm.deviceChecked.push(parseInt(item)) 
					})
				}
				if(data.pciRange){
					vm.settingForm.rangeStart = data.pciRange.split('-')[0];
					vm.settingForm.rangeEnd = data.pciRange.split('-')[1];
				}
				initForm(vm.$refs.settingForm);
			});
		},
		// 设备组选择事件
		deviceGroupSelect(selection){
			var vm = this;
			vm.selection = selection
		},
		// 检测开关改变事件
		detectorSwitchChange(val){
			var vm = this;
			if(val !== '1'){
				vm.settingForm.optimizeSwitch = '0';
			}
		},
		// 优化开关改变事件
		optimizeSwitchChange(val){
			var vm = this;
			if(val !== '1'){
				vm.settingForm.controlledType = '0';
			}
		},
		submit(){ // 确定按钮 
	    	var vm = this;
			vm.$refs.settingForm.validate((valid) => {
				if(valid){
					var params = {
						pciCheckSwitch:vm.settingForm.detectorSwitch,
						pciOptimizeSwitch:vm.settingForm.optimizeSwitch,
						pciControlSwitch:vm.settingForm.controlledType,
						deviceGroupIds:vm.settingForm.cellCodes,
						minMultiplexingDistance:vm.settingForm.reuseDistance,
						pciRange:vm.settingForm.rangeStart + '-' + vm.settingForm.rangeEnd
					};
					axios.post("${ctx}/pci/updateSettings.action",stringify(params)).then(function(response){
						if(response.data.success){
							vm.$message({
                                message:'<%=rb.getString("ChengGong")%>',
                                type:'success',
                            })
                            eventBus.$emit('hide-PCIConfusedSlide');
						}else{
							vm.$message.error(response.data["message"])
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
			if(isFormChanged(this.$refs.settingForm)){
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					eventBus.$emit('hide-PCIConfusedSlide');
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('hide-PCIConfusedSlide');
			}
		},
	},
	
	mounted(){
		this.init();
		eventBus.$off('confusedSetting-ok').$on('confusedSetting-ok',this.submit);
		eventBus.$off('confusedSetting-cancel').$on('confusedSetting-cancel',this.cancel);
	}
})
</script>