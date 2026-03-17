<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#addPciTaskEnb .gridClass .el-input__inner{
		height:26px;
		line-height:26px;
		width:80px;
	}
	#addPciTaskEnb .earfcnClass{
		width:auto;
	}
	#addPciTaskEnb .el-switch__core{
		height:17px;
	}
	#addPciTaskEnb .el-switch__core:after{
		width:13px;
		height:13px;
	}
	#addPciTaskEnb .el-switch.is-checked .el-switch__core::after{
		margin-left:-16px;
	}
	#addPciTaskEnb .errorBorder{
		border:1px solid red;
	}
	#addPciTaskEnb .errorMsg .el-form-item__error{
		left:30px;
	}
	#addPciTaskEnb .queryInfo{
		margin-right:65px;
	}
	#addPciTaskEnb .el-icon-operation-lock:before{
		color:#F2B354;
	}
	#addPciTaskEnb .alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	#addPciTaskEnb .titleStyML{
		margin-left: 20px;
	}
</style>
<div id='addPciTaskEnb' style="margin-top:20px;">
	<el-form ref='enbPciForm' :model="enbPciForm" :rules="rules" label-position="left">
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' style='margin-left:45px;margin-top:20px;' prop='taskName' label-width="120px">
			<el-input v-model='enbPciForm.taskName' maxlength=100 size="mini" style="width:400px;height:28px;line-height:28px;padding-top:5px;"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<el-pairgrid model="normal" 
			style='margin: 20px 45px 0px 45px;'
			@right-load-success='rightLoadSuccess' 
			:id="'pci_select_list'" 
			ref="cpairgrid" 
			@selection-change='selectChange'  
			:right-url="rightUrl" 
			:left-url="leftUrl" 
			:height="height" 
			row-key="small_cell_code" 
			:query-params="queryForm" 
			:title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">
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
				<el-table-column prop='small_cell_code' v-if=false></el-table-column>
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' width="200"></el-table-column>
				<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' width="200" show-overflow-tooltip="true"></el-table-column>
				<el-table-column prop='EARFCNDLINUSE' label='<%=rb.getString("PinDian")%>' width="100">
					<template slot-scope='scope'>
						<div  v-if=scope.row.binding_cpe==1>
							<span class='el-icon el-icon-operation-lock'></span><span v-html='scope.row.EARFCNDLINUSE'></span>
						</div>
						<div v-else>
							<span class='el-icon el-icon-status-unlock'></span><span style='margin-left:5px;' v-html='scope.row.EARFCNDLINUSE'></span>
						</div>
					</template>
				</el-table-column>
				<el-table-column prop='PHYCELLID' label='PCI' width="80">
					<template slot-scope='scope'>
						<div  v-if=scope.row.binding_cpe==1>
							<span class='el-icon el-icon-operation-lock'></span><span v-html='scope.row.PHYCELLID'></span>
						</div>
						<div v-else>
							<span class='el-icon el-icon-status-unlock'></span><span style='margin-left:5px;' v-html='scope.row.PHYCELLID'></span>
						</div>
					</template>
				</el-table-column>
				<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' width='200'></el-table-column>
				<el-table-column prop='Frequency' v-if=false></el-table-column>
			</template>
			<template slot='toolbar'>
				<el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'"
				:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
					<template slot="form">
						<div class='queryInfo'>
							<label><%=rb.getString("XiaoZhanBianMa")%></label>
							<el-input v-model='queryForm.serial_number'  size="mini"></el-input>
						</div>
						<div class='queryInfo'>
							<label><%=rb.getString("HostName")%></label>
							<el-input v-model='queryForm.host_name' size="mini"></el-input>
						</div>
						<div class='queryInfo'>
							<label><%=rb.getString("SheBeiZu")%></label>
							<el-select v-model="queryForm.group_id" size="mini">
								<el-option v-for="item in groupOptions" :key="item.value" :label="item.text" :value="item.value">
								</el-option>
							</el-select>
						</div>
						<div class='queryInfo'>
							<label><%=rb.getString("PinDian")%></label>
							<el-input v-model='queryForm.earfcn'  size="mini"></el-input>
						</div>
					</template>
				</el-query>
			</template>
			<template slot='right'>
				<el-table-column prop='small_cell_code' v-if=false></el-table-column>
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' width='200'></el-table-column>
				<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' width='200' show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("PinDian")%>' width='200'>
					<template slot-scope="scope">
						<el-input class='gridClass earfcnClass' :value=scope.row.EARFCNDLINUSE @blur='checkEarfcn(scope.row.small_cell_code,scope.row.EARFCNDLINUSE,scope.row.PHYCELLID)'></el-input>
	          			<span v-text='"("+scope.row.Frequency+")"'></span>
	          		</template>
				</el-table-column>
				<el-table-column label='PCI' width='120'>
					<template slot-scope="scope">
						<el-input style="width:80px;" class='gridClass' :value=scope.row.PHYCELLID @blur='checkPci(scope.row.small_cell_code,scope.row.PHYCELLID,scope.row.EARFCNDLINUSE)'></el-input>
	          		</template>
				</el-table-column>
				<el-table-column label='Binding CPE' width='120'>
					<template slot-scope="scope">
						<el-switch v-model="switchList[scope.$index].value" @change='changeSwitch(switchList[scope.$index].value,scope.row.small_cell_code)'  active-color='#13ce66' inactive-color='#bbb' ></el-switch>
	          		</template>
				</el-table-column>
			</template>
		</el-pairgrid>
		<el-form-item prop='errorMsg' class='errorMsg' style="margin-left:17px;margin-bottom:0;">
			<el-input v-model='enbPciForm.errorMsg' v-if="false"></el-input>
		</el-form-item>
		<el-form-item prop='cellCodes' class='errorMsg' style="margin-left:45px;margin-bottom:0;">
			<el-input v-model='enbPciForm.cellCodes' v-if="false"></el-input>
		</el-form-item>
		<div style='display:flex'>
			<p style='flex:1 1 60%;'></p>
			<p style='color:#f56c6c;height:20px;margin-left:45px;font-size:13px;flex:1 1 40%;'>{{tipMessage}}</p>
		</div>
		<div class="alarmBottomLine" style="margin-top:20px;"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text">Binding CPEs</span>
		</div>
		<el-ctable style='margin:20px 45px;border:1px solid #E9E9E9;' @load-success='loadSuccessCpe' ref="add_enb_table" :rownumber="true" id="bindCpe_table" :url="bindCpeUrl" height="300px" page-size=20 pagination="true">
			<el-table-column label='eNB SN' width="180" prop='enb_serial_number'></el-table-column>
			<el-table-column label='eNB Name' width="200" prop="host_name"></el-table-column>
			<el-table-column label='<%=rb.getString("CPEBianMa")%>' width="150" prop="cpe_serial_number"></el-table-column>
			<el-table-column label='<%=rb.getString("CPEName")%>' width="200" prop="cpe_host_name"></el-table-column>
			<el-table-column label='<%=rb.getString("PinDian")%>' width="200" prop="cpe_earfcn_before"></el-table-column>
			<el-table-column label='PCI' width="100" prop="cpe_pci_before"></el-table-column>
			<el-table-column label='<%=rb.getString("SheBeiZu")%>' width="200" prop="group_name"></el-table-column>
			<el-table-column label='Earfcn(After)' width='200' prop="cpe_earfcn_after"></el-table-column>
			<el-table-column label='PCI(After)' width='100' prop="cpe_pci_after"></el-table-column>
		</el-ctable>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML" style='margin-top:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<div style='height:60px;border:none;margin-left:30px;margin-top:20px;display:flex;'>
			<el-form-item prop='status'>
				<el-radio-group v-model='enbPciForm.status' style='margin-top:17px;' @change="statusChange">
					<el-radio label='active' style='margin-right:110px;margin-left:20px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
					<el-radio label='suspend' style='margin-right:110px;'><%=rb.getString("GuaQi")%></el-radio>
					<el-radio label='timing'><%=rb.getString("DingShiZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item prop='exetime' class='errorMsg'>
				<el-date-picker v-model="enbPciForm.exetime" style='margin-top:10px;vertical-align:middle;margin-left:25px;' :disabled="setTimeEnable" value-format="yyyy-MM-dd HH:mm:ss" type="datetime"  @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
			</el-form-item>
		</div>
	</el-form>
</div>

<script>
	var addEnb = new Vue({
		el:'#addPciTaskEnb',
		data(){
			var vm = this;
			var validateName = (rule,value,callback) => {
				if(value === ''){
					callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
				}else if(value.trim() == vm.defaultTaskName){
					callback();
				}else{
					axios.post('${ctx}/task/pcilock/taskNameExist.action',stringify({
						taskName:vm.enbPciForm.taskName.trim()
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
			var validateMsg = (rule,value,callback) => {
				var data = vm.$refs.cpairgrid.getData();
				if(data.length == 0){
					callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'));
				}
				if("${type}" == 'edit'){
					setTimeout(function(){
						if(data.length != vm.checkSelect.length){
							callback(new Error('<%=rb.getString("YouWeiXiuGaiXiang")%>'));
						}else{
							vm.checkSelect.map(function(item){
								if(item.earfcnChange || item.pciChange){
								
								}else{
									callback(new Error('<%=rb.getString("YouWeiXiuGaiXiang")%>'));
								}
							})
							callback();
						}
						
					},300)
				}else{
					if(data.length != vm.checkSelect.length){
						callback(new Error('<%=rb.getString("YouWeiXiuGaiXiang")%>'));
					}else{
						vm.checkSelect.map(function(item){
							if(item.earfcnChange || item.pciChange){
							
							}else{
								callback(new Error('<%=rb.getString("YouWeiXiuGaiXiang")%>'));
							}
						})
						callback();
					}
					
					
				}
			};
			var validateTime = (rule,value,callback) => {
				if(this.enbPciForm.status !== 'timing'){
					callback()
				}else{
					if(value == '' || value==null){
						callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
					}else{
						callback();
					}
				}
			};
			return{
				enbPciForm:{
					taskName:'${addTaskName}',
					status:'active',
					errorMsg:'',
					exetime:'',
					cellCodes:''
				},
				rules:{
					taskName:[
						{validator:validateName,trigger:'blur'}
					],
					errorMsg:[
						{validator:validateMsg}
					],
					exetime:[
						{type:'date',validator:validateTime,trigger:'change'}
					]
				},
				leftUrl:'${ctx}/cell/cpeinfos/queryCpeInfosList.action?forSelect=9',
				rightUrl:'',
				queryForm:{
					like_fields:"serial_number,host_name",
					search_text:'',
					CA_FLAG:'0',
					serial_number:'',
					host_name:'',
					group_id:'',
					earfcn:''
				},
				height:'360px',
				deviceTitle:['<%=rb.getString("JiZhanLieBiao")%>','<%=rb.getString("YiXuan")%>'],
				groupOptions:[],
				bindCpe:true,
				bindCpeUrl:'',
				pickerOptions:{
					disabledDate(time){
						return time.getTime()< Date.now()-8.64e7;
					}
				},
				selection:[],
				earfcnError:[],
				earfcn:'',
				tipMessage:'',
				setTimeEnable:true,
				bindCpeGroup:[],
				switchList: [],
				checkSelect:[],
				defaultTaskName:''
			}
		},
		methods:{
			init(){
				var vm = this;
				axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
					selectType : 'deviceGroup'
				})).then(function(response){
					let data = response.data
					vm.groupOptions = data;
				}).catch(function(error){})
			},
			// 根据左侧列表数据 穿梭 获取右侧数据详情
			loadSuccessCpe(){
				var vm = this;
				var rowData = vm.$refs.add_enb_table.getData();
				if(rowData.length > 0 && vm.checkSelect.length > 0){
					rowData.map(function(item){
						vm.checkSelect.map(function(itemCheck){
							if(item.small_cell_code == itemCheck.sn){
								if(itemCheck.earfcnChange){
									item.cpe_earfcn_after = itemCheck.newEarfcnVal
								}
								if(itemCheck.pciChange){
									item.cpe_pci_after = itemCheck.newPciVal;
								}
							}
						})
					})
				}
			},
			/**
			 *select 选择数据
			 * @param selection:所选数据
			*/
			selectChange(selection){
				var vm = this;
				vm.selection = selection;
			},
			// 模糊查询
			query(val){
				this.resetQuery();
				this.queryForm.search_text = val;
				this.$refs.cpairgrid.reload();
			},
			//基站列表 搜索框 高级查询确定按钮
			advanceQuery(){
				this.queryForm.search_text = "";
				this.$refs.cpairgrid.reload();
			},
			//基站列表 搜索框 高级查询重置按钮
			resetQuery(){
				this.queryForm.serial_number = '';
				this.queryForm.host_name = '';
				this.queryForm.group_id = '';
				this.queryForm.earfcn = '';
			},
			/**
			 * 频点input修改
			 * @param code:当前数据code值
			 * @param oldVal:当前数据oldVal值
			 * @param oldPci:当前数据oldPci值
			*/
			checkEarfcn(code,oldVal,oldPci){
				var vm = this,
					target = event.target,
					value = event.target.value,
					reg = /^\d*.?\d{1}$/;
				var snGroup = {
						earfcn:true,
						pci:true,
						earfcnChange:false,
						pciChange:false,
						oldEarfcnVal:oldVal,
						newEarfcnVal:oldVal,
						oldPciVal:oldPci,
						newPciVal:oldPci
				};
				snGroup.sn = code;
				if(value == oldVal){
					snGroup.earfcnChange = false;
				}else{
					snGroup.earfcnChange = true;
				}
				if(!translateToFre(value)){
					target.classList.add('errorBorder');
					vm.tipMessage='<%=rb.getString("FrePCILockAlarm")%>,<%=rb.getString("CiPinDianBuZaiJiZhanZhiChiFanWeiNei")%>';
					target.parentNode.nextElementSibling.innerText = '';
					snGroup.earfcn = false;
				}else{
					value = translateToFre(value);
					var index = value.indexOf("(");
					var indexM = value.indexOf("M");
					var earfcnNumber = value.slice(index+1,indexM);
					var frequencyNumber = value.slice(0,index);
					if (reg.test(earfcnNumber)) {
						if(!(frequencyNumber >= 0 && frequencyNumber <= 65535)){
							target.classList.add('errorBorder');
							target.parentNode.nextElementSibling.innerText = "(" + earfcnNumber + "MHz)";
							vm.tipMessage='<%=rb.getString("FrePCILockAlarm")%>,<%=rb.getString("CiPinDianBuZaiJiZhanZhiChiFanWeiNei")%>';
							snGroup.earfcn = false;
						}else{
							target.classList.remove('errorBorder');
							target.parentNode.nextElementSibling.innerText = "(" + earfcnNumber + "MHz)";
							vm.tipMessage='';
							snGroup.earfcn = true;
							snGroup.newEarfcnVal = frequencyNumber;
							let rowData = vm.$refs.add_enb_table.getData();
							if(rowData.length == 0){
								
							}else{
								rowData.map(function(item){
									if(item.small_cell_code == code){
										item.cpe_earfcn_after = frequencyNumber
									}
								})
							}
						}
					}else{
						target.classList.add('errorBorder');
						target.parentNode.nextElementSibling.innerText = "(" + earfcnNumber + "MHz)";
						vm.tipMessage='<%=rb.getString("FrePCILockAlarm")%>,<%=rb.getString("CiPinDianBuZaiJiZhanZhiChiFanWeiNei")%>';
						snGroup.earfcn = false;
					}
				}
				var codeList = vm.checkSelect.map(function(item){return item.sn});
				if(codeList.includes(code)){
					vm.checkSelect.map(function(item){
						if(item.sn == code){
							item.earfcn = snGroup.earfcn;
							item.earfcnChange = snGroup.earfcnChange;
							item.newEarfcnVal = snGroup.newEarfcnVal;
						}
					})
				}else{
					vm.checkSelect.push(snGroup)
				}
				vm.$refs.enbPciForm.validateField('errorMsg');
			},
			/**
			 * PCI input修改
			 * @param code:当前数据code值
			 * @param oldVal:当前数据oldVal值
			 * @param oldEarfcn:当前数据 oldEarfcn 值
			*/
			checkPci(code,oldVal,oldEarfcn){
				var vm = this,
					target = event.target,
					value = target.value,
					reg = /^\d*$/;
				var snGroup = {
						earfcn:true,
						pci:true,
						earfcnChange:false,
						pciChange:false,
						oldEarfcnVal:oldEarfcn,
						newEarfcnVal:oldEarfcn,
						oldPciVal:oldVal,
						newPciVal:oldVal
				}
				snGroup.sn = code;
				if(value == oldVal){
					snGroup.earfcnChange = false;
				}else{
					snGroup.pciChange = true;
				}
				if(value == ''){
					target.classList.add('errorBorder');
					vm.tipMessage='<%=rb.getString("FrePCILockAlarm")%>,<%=rb.getString("PCILockrange")%>';
					snGroup.pci = false;
				}else{
					if (reg.test(value)) {
						if(!(value >=0 && value <= 503)){
							target.classList.add('errorBorder');
							vm.tipMessage='<%=rb.getString("FrePCILockAlarm")%>,<%=rb.getString("PCILockrange")%>';
							snGroup.pci = false;
						}else{
							target.classList.remove('errorBorder');
							vm.tipMessage='';
							snGroup.pci = true;
							snGroup.newPciVal = value;
							let rowData = vm.$refs.add_enb_table.getData();
							if(rowData.length == 0){
								
							}else{
								rowData.map(function(item){
									if(item.small_cell_code == code){
										item.cpe_pci_after = value;
									}
								})
							}
						}
					}else{
						target.classList.add('errorBorder');
						vm.tipMessage='<%=rb.getString("FrePCILockAlarm")%>,<%=rb.getString("PCILockrange")%>';
						snGroup.pci = false;
					}
				}
				var codeList = vm.checkSelect.map(function(item){return item.sn});
				if(codeList.includes(code)){
					vm.checkSelect.map(function(item){
						if(item.sn == code){
							item.pci = snGroup.pci;
							item.pciChange = snGroup.pciChange;
							item.newPciVal = snGroup.newPciVal;
						}
					})
				}else{
					vm.checkSelect.push(snGroup)
				}
				vm.$refs.enbPciForm.validateField('errorMsg');
			},
			// 此方法暂时没查到
			freFmt(row,column,cellValue,index){
				return translateToFre(cellValue + "");
			},
			/**
			 * 已选择基站 switch 开关
			 * @prame newVal:switch 布尔值
			 * @parame code: codeId
			*/
			changeSwitch(newVal,code){
				var vm = this,
					cellCodes = '';
				if(newVal == false){
					var confirmStr = "<%=rb.getString("DropSelect")%>";
					vm.$alert(confirmStr,'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>'
					})
				}
				vm.switchList.map(function(item){
					if(item.value){
						cellCodes += item.sn + ','
					}
				})
				cellCodes = cellCodes.substring(0,cellCodes.length-1);
				vm.bindCpeUrl = '${ctx}/task/pcilock/getBindingCpeList.action?cellCodes=' + cellCodes +'&operatorCode=' + operator_code;
			},
			// 倒计时
			setTime(){
				this.enbPciForm.exetime = formatDate(new Date(gloableTime));
				this.$refs.enbPciForm.validateField('exetime');
			},
			// 执行方式改变事件
			statusChange(val){
				var vm = this;
				if(val !== 'timing'){
					vm.enbPciForm.exetime = '';
					vm.$refs.enbPciForm.clearValidate('exetime')
				}
			},
			// 确定按钮
			submit(){
				var vm = this,
                    saveUrl = '',
                    saveStr = '';
                // 防止多次提交
                if(pciLockVue.slideSubmitLoading)return

				if(vm.tipMessage != ''){
					return false;
				}
				vm.$refs.enbPciForm.validate((valid) => {
					if(valid){
						var cpeList = vm.switchList.map(function(item){return item.sn})
						vm.checkSelect.map(function(item){
							if(cpeList.includes(item.sn)){
								var index = cpeList.indexOf(item.sn);
								item.bindCpe = vm.switchList[index].value?'1':'0';
							}
						})
						var pciStr = ''
						vm.checkSelect.map(function(item){
							var itemStr = item.sn + '-' + item.oldEarfcnVal + '-' + item.newEarfcnVal + '-' + item.oldPciVal + '-' +
							item.newPciVal + '-' + item.bindCpe;
							pciStr += itemStr + ',';
						})
						pciStr = pciStr.substring(0,pciStr.length-1);
						vm.enbPciForm.errorMsg = pciStr;
						if("${type}" == 'edit'){
							if(isFormChanged(vm.$refs.enbPciForm)){
								saveUrl = '${ctx}/task/pcilock/addTask.action?type=modify&taskId=' + "${taskId}";
								saveStr = '<%=rb.getString("ChengGong")%>'
							}else{
								var submitStr = '<%=rb.getString("WuCanShuBianHua")%>'
								vm.$alert(submitStr,'<%=rb.getString("QueRen")%>',{
									confirmButtonText:'<%=rb.getString("QueDing")%>'
								})
							}
						}else{
							saveUrl = '${ctx}/task/pcilock/addTask.action?type=add';
							saveStr = '<%=rb.getString("ChengGong")%>'
						}
						var params = {};
						params.timeZone = timeZone;
						params.taskName = vm.enbPciForm.taskName;
						params.creator = "${creator}";
						params.status = vm.enbPciForm.status;
						if(vm.enbPciForm.status == 'timing'){
							params.time = vm.enbPciForm.exetime;
						}
						params.pciLock = pciStr;
						vm.enbPciForm.errorMsg = pciStr;
                        pciLockVue.slideSubmitLoading = true;
						axios.post("${ctx}/task/pcilock/validateDoubleCellId.action",stringify(params)).then(function(response){
							if(response.data["message"] && response.data["message"].length > 1){
								showMsg('prompt_msg',"<%=rb.getString("CellIdChongTu")%>");
                                pciLockVue.slideSubmitLoading = false;
							}else{
								var cpeList = vm.checkSelect.map(function(item){return item.bindCpe});
								if(cpeList.includes("1")){
									axios.post("${ctx}/task/pcilock/validateCpeConnectionStatus.action",stringify(params)).then(function(response){
										if(response.data["message"] && response.data["message"].length > 1){
											var confirmStr = response.data["message"] + " <%=rb.getString("CpeBuZaiXianTiShi")%>";
											vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
												customClass:'warningConfirm',
												confirmButtonText:'<%=rb.getString("QueDing")%>',
												cancelButtonText:'<%=rb.getString("QuXiao")%>',
												type:'warning',
												closeOnClickModal:false
											}).then(() => {
												axios.post(saveUrl,stringify(params)).then(function(response){
													if(response.data["success"]){
														vm.$message({
								    						message:saveStr,
								    						type:'success',
								    					})
                                                        eventBus.$emit('save-enb');
													}else{
                                                        pciLockVue.slideSubmitLoading = false;
                                                    }
												})
											}).catch(() => {})
										}else{
											axios.post(saveUrl,stringify(params)).then(function(response){
												if(response.data["success"]){
													vm.$message({
							    						message:saveStr,
							    						type:'success',
							    					})
                                                    eventBus.$emit('save-enb');
												}else{
                                                    pciLockVue.slideSubmitLoading = false;
                                                }
											})
										}
									})
								}else{
									axios.post(saveUrl,stringify(params)).then(function(response){
										if(response.data["success"]){
											vm.$message({
					    						message:saveStr,
					    						type:'success',
					    					})
                                            eventBus.$emit('save-enb');
										}else{
                                            pciLockVue.slideSubmitLoading = false;
                                        }
									})
								}
							}
						})
					}
				})
			},
			
			rightLoadSuccess(rows){
				if("${type}" == 'edit'){
					var vm = this;
					rows.map(function(item){
						item.PHYCELLID = item.pci;
						item.EARFCNDLINUSE=item.earfcn;
						item.Frequency = item.freq + "(MHz)";
					})
					
					var data = rows;
					var newList = [];
					data.map(function(item){
						var obj = {
								sn : item.small_cell_code,
								value : item.cpe_flag == '1' ? true : false
						}
						newList.push(obj);
						var strObj = {
							sn : item.small_cell_code,
							earfcn:true,
							pci:true,
							earfcnChange : true,
							pciChange : true,
							oldEarfcnVal : item.earfcn,
							newEarfcnVal : item.earfcn,
							oldPciVal : item.pci,
							newPciVal : item.pci,
							bindCpe:item.cpe_flag
						}
						vm.checkSelect.push(strObj);
					});
					vm.switchList = newList;
					var pciStr = ''
					vm.checkSelect.map(function(item){
						var itemStr = item.sn + '-' + item.oldEarfcnVal + '-' + item.newEarfcnVal + '-' + item.oldPciVal + '-' +
						item.newPciVal + '-' + item.bindCpe;
						pciStr += itemStr + ',';
					})
					pciStr = pciStr.substring(0,pciStr.length-1);
					vm.enbPciForm.errorMsg = pciStr;
					vm.bindCpeUrl = '${ctx}/task/pcilock/getBindingCpeList.action?cellCodes=' + vm.enbPciForm.cellCodes +'&operatorCode=' + operator_code + "&taskId=" + "${taskId}";
					setTimeout(function(){
    					initForm(vm.$refs.enbPciForm);
    				},500);
				}
			},
			// 当前基本信息
			getEnbInfo(){
				var vm = this;
				//获取基本信息
				axios.post('${ctx}/task/pcilock/getEnbPciLockTaskInfo.action?taskId='+"${taskId}"+"&timeZone="+timeZone).then(function(response){
					let data = response.data;
					vm.enbPciForm.taskName = data.taskName;
					vm.defaultTaskName = data.taskName;
					vm.enbPciForm.status = data.executeType;
					if(data.quartzTime == null || data.quartzTime == ''){
						vm.enbPciForm.exetime = '';
					}else{
						vm.enbPciForm.exetime = data.quartzTime;
					}
				})
				//获取已经选择的enb设备列表
				vm.rightUrl = '${ctx}/task/pcilock/getSelectedList.action?taskId='+"${taskId}";
			},
			// 关闭新建窗口
			cancelSubmit(){
				var vm = this;
				var cpeList = vm.switchList.map(function(item){return item.sn})
				vm.checkSelect.map(function(item){
					if(cpeList.includes(item.sn)){
						var index = cpeList.indexOf(item.sn);
						item.bindCpe = vm.switchList[index].value?'1':'0';
					}
				})
				var pciStr = ''
				vm.checkSelect.map(function(item){
					var itemStr = item.sn + '-' + item.oldEarfcnVal + '-' + item.newEarfcnVal + '-' + item.oldPciVal + '-' +
					item.newPciVal + '-' + item.bindCpe;
					pciStr += itemStr + ',';
				})
				pciStr = pciStr.substring(0,pciStr.length-1);
				vm.enbPciForm.errorMsg = pciStr;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
				if(isFormChanged(vm.$refs.enbPciForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						eventBus.$emit('close-enb');
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('close-enb');
				}
			}
		},
		watch:{
			"enbPciForm.status":function(newVal){
				if(newVal == 'timing'){
					this.setTimeEnable = false
				}else{
					this.setTimeEnable = true
				}
				this.$refs.enbPciForm.validateField('exetime');
			},
			selection(){
				var vm = this;
				vm.$refs.enbPciForm.validateField('errorMsg');
				var data = vm.$refs.cpairgrid.getData();
				/* 已选设备改变时获取是否绑定cpe */
				var oldmap = {},
					newList = [];
				vm.switchList.map(function(item){
					oldmap[item.sn] = item.value;
				});
				data.map(function(row){
					var sn = row.small_cell_code,
						snlist = vm.switchList.map(function(item){return item.sn;}),
						bool = true;
					if(snlist.includes(sn)) {
						bool = oldmap[sn];
					}
					newList.push({sn: sn, value: bool});
				});
				vm.switchList = newList;
				var cellCodes = '';
				vm.switchList.map(function(item){
					if(item.value){
						cellCodes += item.sn + ','
					}
				})
				cellCodes = cellCodes.substring(0,cellCodes.length-1);
				vm.enbPciForm.cellCodes = cellCodes;
				vm.bindCpeUrl = '${ctx}/task/pcilock/getBindingCpeList.action?cellCodes=' + cellCodes +'&operatorCode=' + operator_code;
				/* 当数据改变时改变校验数组 */
				if(data.length == 0){
					vm.checkSelect = []
				}else{
					var checkList = data.map(function(item){return item.small_cell_code});
					var earAndPciList = vm.checkSelect.map(function(item){return item.sn});
					vm.checkSelect.map(function(item){
						if(checkList.includes(item.sn)){
							
						}else{
							var index = earAndPciList.indexOf(item.sn);
							vm.checkSelect.splice(index,1);
						}
					})
				}
			}
		},
		mounted(){
			this.init();
			eventBus.$off('add-enb').$on('add-enb',this.submit);
			eventBus.$off('get-enb-info').$on('get-enb-info',this.getEnbInfo);
			eventBus.$off('cancel-add-enb').$on('cancel-add-enb',this.cancelSubmit);
		}
	})
</script>