<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#addViewTracePage .el-icon-time{
	line-height:1;
}

#addViewTracePage .timeItem .el-input__inner{
	width:220px;
}
#addViewTracePage .ml45{
	margin-left: 66px
}
#addViewTracePage .mt16{
	margin-top: 16px
}
#addViewTracePage .taskInput{
	width:300px;
	height:30px;
	line-height:30px;	
}
#addViewTracePage .radioBox{
	border:1px solid #D1ECF5 !important;
	width:660px;
	padding:15px 0px 15px 0px;
}
#addViewTracePage .el-form-item__label{
	line-height:26px;
	width:140px;
	text-align:left;
}

#addViewTracePage .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
#addViewTracePage .titleStyML{
	margin-left: 40px;
}
#addViewTracePage .el-pairgrid-title{
	top:-6px;
}
#addViewTracePage .deviceItem{
	display:inline-block;
	margin-right:60px;
}
#addViewTracePage .pairgrid-right .el-ctable-toolbar{
	padding: 10px!important;
}
#addViewTracePage .commonBorder { border: 1px solid #E9E9E9; border-radius: 10px;}
#addViewTracePage .commonToolBarBox .el-query .advanceQuery .el-icon { font-size: 14px; }
#addViewTracePage .commonFontSize12 { font-size: 12px; }
#addViewTracePage .commonColor, 
#addViewTracePage .commonToolBarBox .el-query .advanceQuery .el-icon:before { color: #7A7992; }
#addViewTracePage .commonColor2 {  color: #999999; }
#addViewTracePage .commonToolBarBox { height: 30px; }
#addViewTracePage .marginLeft5 { margin-left: 5px; }
#addViewTracePage .el-form-item { margin-bottom: 22px; }
#addViewTracePage .el-form-item__label { color: #666666; } 
#addViewTracePage .el-radio.is-bordered { max-width: 200px; height: 30px; padding: 7px 12px; }
#addViewTracePage .el-checkbox.is-bordered { max-width: 200px; height: 30px; padding: 6px 12px; }
#addViewTracePage .el-radio-group .el-radio__label, 
#addViewTracePage .el-checkbox-group .el-checkbox__label { font-size: 12px; line-height: unset;}
#addViewTracePage .el-input__inner { height: 30px; }
#addViewTracePage .selected-status:before { color: #67D972 !important; }
#addViewTracePage .textareStyle .el-textarea__inner { resize: none; border-radius: 4px; }
#addViewTracePage .el-input__inner { border-radius: 4px; }
</style>

<div id="addViewTracePage" style="margin-top:20px;">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" label-position="top">
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text commonColor"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<div class='commonFlex mt16'>
			<el-form-item label='<%=rb.getString("GenZongMingCheng")%>' prop='traceName' class="ml45">
				<el-input maxlength=100  v-model="ruleForm.traceName" class="taskInput" style='width: 330px;'></el-input>
				<span class='commonFontSize12 commonColor2 marginLeft5'><%=rb.getString("ChangDuXianZhi")%> 1-100</span>
			</el-form-item>
			<el-form-item label='<%=rb.getString("GenZongCanKaoHao")%>' prop='traceId' style='margin-left: 80px;'>
				<el-input v-model="ruleForm.traceId" size="mini" disabled></el-input>
			</el-form-item>
		</div> 
		<div class='commonFlex ml45' >
			<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='remarks' style='margin: -5px 0 30px; '>
                <el-input v-model='ruleForm.remarks' type='textarea' :rows='3' class="textareStyle" style='resize: none;width: 516px;'></el-input>
            </el-form-item> 
      	</div>
		
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text commonColor"><%=rb.getString("GenZongPeiZhi")%></span>
		</div>
		<div class='commonFlex ml45 mt16'>
			<el-form-item prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>">
				<el-select v-model='ruleForm.product' @change='productChange' style='width: 230px;'>	
					<el-option v-for='item in productDataList' :label="item.name" :value="item.value"></el-option>
				</el-select>
			</el-form-item>
		</div>		
		<el-ctable :id="'select_device_list'" :row-key="'serial_number'" ref="deviceCtable" :url='deviceListUrl'
		 	@row-click="rowClickDevice" :height="height" pagination="true" :query-params="deviceParams" class='commonBorder' style='margin: 0 66px;'>
			<div slot="toolbar">
				<div class='commonFlex commonToolBarBox commonContent licenseBox' style="justify-content: space-between;">
					<span class='commonFontSize12 commonColor2' style='padding: 6px 20px;'><%=rb.getString("SheBeiLieBiaoBiaoTi")%></span>
					<div class='commonFlex'>
						<el-query type="normal" @query="queryDevice" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>" style='margin-right: 10px;'></el-query>
					</div>
				</div>
			</div>
			<el-table-column width="50">
				<div slot-scope="scope" style="margin: 0 auto;">
					<el-radio v-model="caFileId" :label="scope.row.serial_number"><span></span></el-radio>
				</div>
			</el-table-column>
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
		</el-ctable>

		<el-form-item prop='traceDevice' style="margin-left:64px;margin-bottom:30px;">
			<el-input v-model='ruleForm.traceDevice' v-show="false"></el-input>
		</el-form-item>
		<div class='commonFlex ml45 mt16'>

			<el-form-item label="<%=rb.getString("JieKouLeiXing") %>" prop="NEInterface" style='margin-bottom: 30px;' class='commonFlex'>
                <el-checkbox-group v-if="!isGnb" v-model="ruleForm.NEInterface" >
                    <el-checkbox label="S1" border>S1 <%=rb.getString("HE")%> X2</el-checkbox>
                    <el-checkbox label="Uu" border style='margin-left: 20px;'>Uu</el-checkbox>
                </el-checkbox-group>

                <el-checkbox-group v-if="isGnb" v-model="ruleForm.NEInterface" >
                    <el-checkbox label="NG" border>NG</el-checkbox>
                    <el-checkbox label="F1" border style='margin-left: 20px;'>F1</el-checkbox>
                    <el-checkbox label="XN" border style='margin-left: 20px;'>XN</el-checkbox>
                </el-checkbox-group>
            </el-form-item>
		</div>
		
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text commonColor"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<div class='commonFlex mt16'>
			<el-form-item label=" " prop="executeMode" class="ml45">
	            <el-radio-group v-model="ruleForm.executeMode" @change="statusChange">
	                <el-radio label="1" border><%=rb.getString("LiJiZhiXing") %></el-radio>
	                <el-radio label="2" border style='margin-left: 20px;'><%=rb.getString("GuaQi") %></el-radio>
	                <el-radio label="3" border style='margin-left: 20px;'><%=rb.getString("DingShiZhiXing") %></el-radio>
	            </el-radio-group>
	        </el-form-item>
			<el-form-item prop='exetime' style='display:inline-block;margin: 6px 10px 0;' class='timeItem'>
				<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="setTimeEnable" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
			</el-form-item>
		</div>
		
		<div>
			<el-form-item label='<%=rb.getString("ChiXuShiChang")%>' prop='duration' class="ml45" style='margin-bottom: 30px;'>
				<el-input maxlength=50  v-model="ruleForm.duration" size="mini" class="taskInput"></el-input>
				<span class='commonFontSize12 commonColor2 marginLeft5'><%=rb.getString("ZhengXing")%>[1:30],<%=rb.getString("DanWei")%>:<%=rb.getString("FenZhongDaXie")%></span>
			</el-form-item>
		</div>
	</el-form>
	
</div>

<script type="text/javascript">
var addViewTraceVue = new Vue({
	el:'#addViewTracePage',
	data(){
		var vm = this,
		validateName = (rule,value,callback) => {
			if(value === '' || value === null){
				callback(new Error('<%=rb.getString("GenZongMingChengBuNengWeiKong")%>'))
			}else{
				callback();
			}
		},
		validateTraceDevice = (rule,value,callback) => {
			var value = vm.caFileId;
			if(value === '' || value === null){
            	callback('<%=rb.getString("QingXuanZeGenZongSheBei") %>');
            }else{
            	callback();
            }
		},
		validateTime = (rule,value,callback) => {
			if(vm.ruleForm.executeMode !== '3'){
				callback()
			}else{
				if(value === '' || value === null){
					callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
				}else{
					callback();
				}
			}
		},
		validateNEInterface = (rule,value,callback) => {
			if(vm.ruleForm.NEInterface.length == 0) {
            	callback('<%=rb.getString("BiTian") %>');
            }else{
            	callback();
            }
		},
		validateDuration = (rule,value,callback) => {
			var reg = /^[0-9]+$/;
			if(value === '' || value === null){
				callback(new Error('<%=rb.getString("XinLingZhuiZongZuiDaShiChang")%>'))
			}else if(reg.test(value) && (value >= 0) && (value <= 30)){
				callback();
			}else{
				callback(new Error('<%=rb.getString("XinLingZhuiZongZuiDaShiChang")%>'))
			}
		};
		
		return {
			isGnb: sysMain.headType == 'gnb',
			height:'270px',
			setTimeEnable: true,
			ruleForm:{
				traceName:'${taskName}',
				traceId: '${traceId}',
				remarks: '',
				NEInterface: [],
				traceDevice:'',
				executeMode:'1',
				exetime:'',
				duration:'',
				product: ''
			},
			rules:{
				traceName:[ {validator: validateName,trigger:'blur'} ],
				traceDevice:[ {validator: validateTraceDevice,trigger:'change'} ],
				NEInterface: [{validator: validateNEInterface,trigger:'blur'}],
				exetime:[ {type:'date',validator: validateTime,trigger:'change'} ],
				duration: [{validator: validateDuration,trigger:'blur'}]
			},
			queryParams:{
				timeZone:timeZone,
				isShowSlave:false,
				isShowRTDDC:'true',
				serial_nubmer:'',
				host_name:'',
				isGnb: sysMain.headType == 'gnb'? 1:'',
				search_text:'',
				group_id:'',
			},
			queryForm:{
				serial_nubmer:'',
				host_name:'',
				group_id:'',
			},
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			productDataList:[],
			deviceListUrl: '',
			//deviceListUrl:'${ctx}/signaling/queryENBTraceDevice.action',
			deviceParams:{
				timeZone:timeZone,
				likeFields:'host_name,serial_number',
				searchText: '',
				product_type: '',
				isGnb: sysMain.headType == 'gnb'? 1:'',
			},
			rowData : [],
			productType:'',
			curType: '',
			caFileId:'',
		}
	},
	watch:{
		rowData(newVal){
			var vm = this;
			vm.caFileId = newVal.serial_number;
			vm.ruleForm.traceDevice = newVal.serial_number;
		},
		"ruleForm.NEInterface":function(newVal){
			if(newVal.length > 0){
				this.$refs.ruleForm.clearValidate('NEInterface')
			}else{
				this.$refs.ruleForm.validateField('NEInterface')
			}
		},
		"ruleForm.executeMode":function(newVal){
			if(newVal == '3'){
				this.setTimeEnable = false
			}else{
				this.setTimeEnable = true
			}
			this.$refs.ruleForm.validateField('exetime')
		},
		"ruleForm.product":function(newVal){
			
			this.deviceParams.product_type = newVal;
			this.deviceListUrl = '${ctx}/signaling/queryENBTraceDevice.action';
		}
	},
	methods:{ 
		
		init(type, row){
			var vm = this;	

			var prodUrl = '${ctx}/cell/cpeinfos/getEnbMonitorProductList.action?isGnb=0';
			if(vm.isGnb) {
				prodUrl = '${ctx}/cell/cpeinfos/getEnbMonitorProductList.action?isGnb=1'
			}
			//产品类型
			axios.post(prodUrl).then(function(response){
		   		var data = response.data;
		   		if(data.length > 0){
		   			vm.productDataList = data.map(function(item){
			   			if (item){
			   				return {name:item,value:item}
			   			}
			   		})
			   		//取数组第一个为默认
			   		vm.ruleForm.product = vm.productDataList[0].value;
		   		}else{
			   		vm.ruleForm.product = '';
		   		}
		   })		   
		},
		productChange(val){
			var vm = this;
			//vm.deviceParams.product_type = val;		
		},
		queryDevice(text) {
			this.deviceParams.searchText = text;
		},
		
		rowClickDevice(row){
			var vm = this;
			//vm.caFileId = row.serial_number;
			//vm.ruleForm.traceDevice = row.serial_number;
			vm.rowData = row;
	    },
		// 执行方式改变事件
		statusChange(val){
			var vm = this;
			if(val !== '3'){
				vm.ruleForm.exetime = '';
				vm.$refs.ruleForm.clearValidate('exetime')
			}
		},
		submit(){ // 确定按钮 
	    	var vm = this, url = '${ctx}/signaling/addENBSignalingTraceTask.action', message = '',
				params = {
					traceName: vm.ruleForm.traceName,
					traceId: vm.ruleForm.traceId,
					timeZone: timeZone,
					remarks: vm.ruleForm.remarks,
					traceNetworkElement: vm.isGnb?'gNB':'eNB',
					NEInterface: vm.ruleForm.NEInterface.join(','),
					identificationInfo: "SN"+"="+vm.ruleForm.traceDevice, // sn
					executeMode: vm.ruleForm.executeMode,
					duration: vm.ruleForm.duration
				};
	    	
			var deviceName = vm.rowData.host_name?vm.rowData.host_name:'';
    		var deviceSn = vm.ruleForm.traceDevice;
    		var showVal = deviceName+'('+deviceSn+')';
    		params.traceDevice = showVal;
	    	if(vm.ruleForm.executeMode == '3'){
				params.startTime = vm.ruleForm.exetime;
			}
            // 防止多次提交
            if(traceVue.slideSubmitLoading)return
            
	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
                    traceVue.slideSubmitLoading = true;
                    axios.post(url,stringify(params)).then(function(response){
                        var data = response.data;
                        if(data["success"]){
                            vm.$message({
                                message:'<%=rb.getString("ChengGong")%>',
                                type:'success'
                            })
                            eventBus.$emit('hander-cancel');
                        }else{
                            vm.$message.error(data["message"]);
                            traceVue.slideSubmitLoading = false;
                        }
                    })
	    		}
	    	})
		},
		// 关闭弹窗
		closeSlider(){ 
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
			if(isFormChanged(vm.$refs.ruleForm)){
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					eventBus.$emit('hander-cancel');
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('hander-cancel');
			}
		},
		setTime(){
			var vm = this;
			vm.ruleForm.exetime = formatDate(new Date(gloableTime));
			vm.$refs.ruleForm.validateField('exetime');
		},
		
	},
	
	mounted(){
		eventBus.$off('init-config').$on('init-config', this.init); 
		eventBus.$off('hander-ok').$on('hander-ok',this.submit);
		eventBus.$off('close-slide').$on('close-slide',this.closeSlider);
	}
})
</script>
