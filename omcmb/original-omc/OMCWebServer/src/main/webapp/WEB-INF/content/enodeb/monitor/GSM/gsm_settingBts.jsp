<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#gsmSettingBtsPage{
	height: 100%;
	width: 100%;
}
#gsmSettingBtsPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#gsmSettingBtsPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #E9E9E9;
}
#gsmSettingBtsPage .itemMainBoxCenter{
	width: 100%;
	flex:1;
	overflow: auto;
}
#gsmSettingBtsPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#gsmSettingBtsPage .rightContentCls .contentTableTitle{
	display: flex;
	justify-content: space-between;
	font-weight: 550;
	width: 100%;
}
#gsmSettingBtsPage .rightContentCls .contentTableTitle>div:nth-child(1){
	font-size: 12px;
}
#gsmSettingBtsPage .moreIpItemBoxCls{
	display: flex;
	flex-wrap: wrap;
	width: 100%;
}
#gsmSettingBtsPage .leftAndRightItemCls{
	width:40%;
	min-width:400px;
	margin-bottom: 20px;
}
#gsmSettingBtsPage .itemListBoxCls{
	padding-top: 5px;
}
#gsmSettingBtsPage .itemCls{
	height: 24px;
	display: inline-block;
	line-height: 24px;
	border: 1px solid #4D84FF;
	box-sizing: border-box;
	padding: 0px 10px;
	margin-right: 10px;
	margin-bottom: 10px;
}
#gsmSettingBtsPage .itemListBoxCls .el-icon-close{
	font-size: unset;
	position: unset;
	top: unset;
	right: unset;
}
#gsmSettingBtsPage .leftAndRightItemCls .el-input__suffix{
	height: 26px;
	display: flex;
	align-items: center;
}
#gsmSettingBtsPage .el-form-item{
	margin-bottom: 20px;
}
#gsmSettingBtsPage .errorBoxCls{
	color:red;
	font-size:10px;
}
#gsmSettingBtsPage .multiPlmnEnableBoxCls .el-form-item__label{
	padding-top: 13px;
	margin-right: 20px;
}
#gsmSettingBtsPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
#gsmSettingBtsPage .itemMainBoxCls .itemMainBoxCenter .el-collapse-item__wrap{
    border-bottom:none;
    padding-left:40px;
}
.gsmConfigAddDialog .gsmConfigAddMainBoxCls{
	height: 400px;
	width: 100%;
	position: relative;
	overflow-y: auto!important;
	overflow-x:hidden;
}
.gsmConfigAddDialog .el-dialog__header .el-icon:before{
	font-size: 16px;
}
</style>

<div id="gsmSettingBtsPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">
			BTS
			<!-- 按钮  同步 -->
			<div class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="syncSettingsClick" tip="<%=rb.getString("TongBu")%>">
				<span class="el-icon el-icon-circle-refresh"></span>
			</div>
		</div>
		<div class="itemMainBoxCenter">
			<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="top">
				<el-collapse v-model="activeCollapse">
					<el-collapse-item name="BTSInfo">
						<template slot='title'>
							<p style="display:inline-block;margin-left:40px;">
								<span style="font-size:14px;font-weight:bold">BTS Info</span>
							</p>
						</template>
						<div class="rightContentCls" >
							<!--AMF-->
							<div> 
								<div class="cellTableBoxCls" style="padding-bottom:20px;margin-right:60px;">
									<el-ctable
										ref="gsmBtsInfoTable" 
										:row-class-name="tableRowClassName"
										:rownumber="true" 
										id="gsmBtsInfoTable" 
										:data="ruleForm.BTSList"
										height="500px"
										:pagination="true"
										:front-pagination="true"
										style="border:1px solid #E9E9E9;"
									>
										<el-table-column label='' width="40">
											<template slot-scope="scope">
												<span class="el-icon el-icon-operation-edit" @click="settingsBTSList(scope.row,event)" ></span>
											</template>
										</el-table-column>
										<el-table-column label='<%=rb.getString("BTSBianMa")%>' min-width="120" prop="BTS_SerialNumber" show-overflow-tooltip></el-table-column>
										<el-table-column label='<%=rb.getString("SheBeiXingHaoMing")%>' min-width="120" prop="BTS_ModelName" show-overflow-tooltip></el-table-column>
										<el-table-column label='CellIdentity' min-width="120" prop="BTS_CellIdentity" show-overflow-tooltip></el-table-column>
										<el-table-column label='<%=rb.getString("ShiFouJiHuo")%>' min-width="120" prop="BTS_CellStatus" show-overflow-tooltip>
											<template slot-scope="scope">
												<div v-if="scope.row.BTS_CellStatus == '1'" class='iconFlexCls'>
													<span style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
												</div>
												<div v-if="scope.row.BTS_CellStatus == '0'" class='iconFlexCls'>
													<span class='offStatusCls' style='margin-right: 5px;'><%= rb.getString("QuJiHuo")%></span>
												</div>
											</template>
										</el-table-column>
										<el-table-column label='<%=rb.getString("SoftwareVersion")%>' min-width="120" prop="BTS_SoftwareVersion" show-overflow-tooltip></el-table-column>
									</el-ctable>
									<el-form-item prop='BTSList' style="display:none;" label="" label-width="0px">
										<el-input v-model='ruleForm.BTSList'></el-input>
									</el-form-item>
								</div>
							</div>
						</div>
					</el-collapse-item>
				</el-collapse>
			</el-form>
		</div>
		<div class='itemMainBoxFooter'>
			<el-button type="primary" @click="settingsSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSettings" ><%=rb.getString("QuXiao")%></el-button>
		</div>
		<!-- slide -->
		<el-slide  ref="sharingSlide" :url='sharingSlideUrl' :title="sharingSlideTitle" :footer="sharingSlideFooter" :header="sharingSlideHeader" :position="sharingSlidePosition"
			:height="sharingSlideHeight" :modal='modal' :width='sharingSlideWidth'  @cancel="sharingSlideCancel" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
	</div>
	<!-- BTS Info 修改 弹窗 -->
	<el-dialog class="gsmConfigAddDialog" top="25vh" title="<%=rb.getString("XiuGai")%>" width="50%" :visible.sync="addBtsInfoDialogShow" @close="closeAddBtsInfoDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addBtsInfoDialogForm" :model='addBtsInfoDialogForm' :rules='addBtsInfoDialogRules' label-position="top">     		     			            
			<el-form-item prop='BTS_SerialNumber' style="min-width:400px;" label="<%=rb.getString("BTSBianMa")%>" label-width="160px">
				<el-input v-model.trim='addBtsInfoDialogForm.BTS_SerialNumber' :disabled="true"></el-input>
			</el-form-item>
			<el-form-item prop='BTS_ModelName' style="min-width:400px;" label="<%=rb.getString("SheBeiXingHaoMing")%>" label-width="160px">
				<el-input v-model.trim='addBtsInfoDialogForm.BTS_ModelName' :disabled="true"></el-input>
			</el-form-item>
			<el-form-item prop='BTS_CellIdentity' style="min-width:400px;" label="CellIdentity" class='validate-item'>
				<el-input v-model.trim='addBtsInfoDialogForm.BTS_CellIdentity'>
					<template slot="append"><%=rb.getString("FanWei")%>：0~65535,Integer</template>
				</el-input>
			</el-form-item>
			<el-form-item prop='BTS_CellStatus' style="min-width:400px;" label="<%=rb.getString("ShiFouJiHuo")%>" label-width="160px">
				<el-select v-model='addBtsInfoDialogForm.BTS_CellStatus' :disabled="true">
					<el-option label='<%= rb.getString("JiHuo")%>' value='1'></el-option>
					<el-option label='<%= rb.getString("QuJiHuo")%>' value='0'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='BTS_SoftwareVersion' style="min-width:400px;" label="<%=rb.getString("SoftwareVersion")%>" label-width="160px">
				<el-input v-model.trim='addBtsInfoDialogForm.BTS_SoftwareVersion' :disabled="true"></el-input>
			</el-form-item>
		</el-form> 
		<div slot="footer" class="importFooter">
			<el-button type="primary" @click="addBtsInfoDialogSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="addBtsInfoDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>			
	</el-dialog>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var gsmSettingBtsPageVue = new Vue({
	el: '#gsmSettingBtsPage', 
	data() {
		var vm = this,
			validateRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var isRequired = rule.isRequired;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

				if(value == '' || value == undefined || value == null){
					if(isRequired){
						callback(new Error(mag))
					}else{
						callback();
					}
				}else{
					if(reg.test(value) && value >= min && value <= max){
						callback();
					}else{
						callback(new Error(mag))
					}
				}
			};
		return {
			activeCollapse:['BTSInfo'],
			rowDataInfo: [],
			smallCellCode:'',
			ruleForm:{
				BTSList:[],
			},
			rules:{
				
			},
			casts:{
				'570E1CCD74C31591036983DE09FEE0E9':'BTSList',
				'342690659115822166E1B3B8CFE8EAC1':'BTS_idx',
				'60DA6B50747A31938F46626E4446610B':'BTS_SerialNumber',
				'A7B93E6D7E7422F3C2D1F2B23E01319B':'BTS_ModelName',
				'25D3DF6F80998148616BAC44DC47EE2F':'BTS_CellIdentity',
				'DA9812D9B0A521BF5F3DE452372FC7E8':'BTS_CellStatus',
				'06A8447715BC6FC092C8BFCDAED5B31F':'BTS_SoftwareVersion',
				'80EA88A557A2FF69FCF7084B65325FE9':'BTS_NumOfTrxChannel',
			},
			codeList:[],
			optType:'',
			tbType:'',

			sharingSlideUrl:'',
			sharingSlideTitle:'',
			sharingSlideFooter:'',
			sharingSlideHeader:'',
			sharingSlidePosition:'',
			sharingSlideHeight:'',
			sharingSlideWidth:'',

			addBtsInfoDialogShow:false,
			addBtsInfoDialogForm:{
				BTS_SerialNumber:'',
				BTS_ModelName:'',
				BTS_CellIdentity:'',
				BTS_CellStatus:'',
				BTS_SoftwareVersion:'',
			},
			addBtsInfoDialogRules:{
				BTS_CellIdentity:[{validator:validateRange,min:0,max:65535,mag:'<%=rb.getString("FanWei")%>：0~65535,Integer',isRequired:true}],
			}
		};
	},
	computed: {
		
	},
	methods: {
		init(row){
			var vm = this;
			vm.rowDataInfo = row;
			vm.smallCellCode = row.small_cell_code;
			var codeList=[];
			Object.keys(vm.casts).forEach(function(key){
				codeList.push(key)
			});
			vm.codeList = codeList;
			vm.getParamData(vm.smallCellCode,'23003');
		},
		getParamData(code,id) {
			var vm = this,
				codes = [],
				url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
				params = {
					id: id,
					smallCellCode: code
				};
			axios.post(url, stringify(params)).then(function(res){
				var data = res.data;
				vm.resetFormData();
				if(data && Array.isArray(data)) {
					data.map(function(item){
						item.groups.map(function(group){
							group.list.map(function(m){
								if(m.type == 'list'){
									vm.initTable(m.url,m.label);
								}else{
									codes.push(m.name);
									// 执行赋值
									vm.setValue(m);
								}
							});
						});
					});
					initForm(vm.$refs.ruleForm);
				}
			});
		},
		// 映射赋值
		setValue(item) {
			var vm = this,
			code = item.name,
			value = item.value;

			// indexs是否含有
			var key = vm.casts[code];
			try{
				if(key){
					vm.ruleForm[key] = value;
				}
			}catch(e){}
		},
		initTable(url,type){
			var vm = this,codes = [];
			var params = {
					smallCellCode : vm.smallCellCode
			}
			axios.post(url,stringify(params)).then(res=>{
				var data = res.data;
				if(data.rows){
					data.rows.map(item=>{
						var obj = {};
						for(var key in item){
							codes.push(key);
							obj[vm.casts[key]] = item[key]
						}
						if(type == 'BTS List'){
							vm.ruleForm.BTSList.push(obj);
						}
					})
					initForm(vm.$refs.ruleForm);
				}
			})
		},
		// 重置form数据
		resetFormData(){
			var vm =this;
				params={
					BTSList:[],
				};
			Object.assign(vm.ruleForm,params);
		},
		// BTS Info 修改
        settingsBTSList(row){
            var vm = this;

			/*vm.sharingSlideUrl = '${ctx}/cell/cpeinfos/toGSMMonitorSettingBtsDetailPages.action';
			vm.sharingSlideHeight = '100%';
			vm.sharingSlideWidth = '100%';
			vm.sharingSlideFooter = false;
			vm.sharingSlidePosition = 'top';
			vm.sharingSlideHeader = false;
			vm.sharingSlideTitle = '';
			vm.$refs.sharingSlide.showSlide(function(){
				eventBus.$emit('btsDetails-init',row,vm.smallCellCode);
			});*/
			Object.assign(vm.addBtsInfoDialogForm,row);
			vm.addBtsInfoDialogShow = true;
        },
		// BTS Info 修改 提交
		addBtsInfoDialogSubmit(){
			var vm = this,
				params = {},
				idxStr = 'BTS_idx',
				optTb = 'BTSList';
			Object.keys(vm.addBtsInfoDialogForm).forEach(function(key){
				params[key] = vm.addBtsInfoDialogForm[key]
			})
			if(vm.addBtsInfoDialogForm.operateType){
				params.operateType = vm.addBtsInfoDialogForm.operateType
			}
			vm.$refs.addBtsInfoDialogForm.validate(function(valid){
				if(valid){
					if(!params.operateType){
						params.operateType = 'edit';
					}
					var idx='';
					vm.ruleForm[optTb].map((item,index)=>{
						if(item[idxStr] == params[idxStr]){
							idx = index
						}
					})
					Object.assign(vm.ruleForm[optTb][idx],params)
					vm.addBtsInfoDialogShow = false;
				}
			})
		},
		// 关闭 PCD 弹窗
		closeAddBtsInfoDialog(){
			var vm = this,
				params = {
					BTS_SerialNumber:'',
					BTS_ModelName:'',
					BTS_CellIdentity:'',
					BTS_CellStatus:'',
					BTS_SoftwareVersion:'',
				};
			Object.assign(vm.addBtsInfoDialogForm,params);
			vm.$refs.addBtsInfoDialogForm.clearValidate();
		},
		// 关闭slide页面
		sharingSlideCancel(){
			this.$refs.sharingSlide.hide();
		},
		// 判断是否为空
		isNull(val){
			if(val==undefined || val == null || val =="") return true;
			else return false;
		},
		// 验证输入的是否是整数
		isInteger(str) {
			if(str.length==0){
				return false;
			}
			var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
			if(!reg.test(str)){
				return false;
			}
			return true;  
		},
		tableRowClassName({row,rowIndex}){
			if(row.operateType && row.operateType == 'remove'){
				return 'hidden-row'
			}
			return ''
		},
		settingsSubmit(){
			var vm = this;
			var params = {},
				isChanged = isFormChanged(vm.$refs.ruleForm),
				isSync = false;

			if(!isChanged){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
				return;
			}

			vm.$refs.ruleForm.fields.map(function(field){
				var key = vm.getNameByProp(field.prop);

				if(Array.isArray(field.fieldValue)){
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						val = JSON.stringify(vList.sort()),
						orVal = JSON.stringify(oList.sort());

					if(val != orVal) {
						var editList=[],subList=[];
						vList.map((items)=>{
							if(items.operateType){
								var subObjs={};
								subObjs.BTS_idx = items.BTS_idx;
								subObjs.BTS_CellIdentity = items.BTS_CellIdentity;
								subObjs.operateType = items.operateType;
								editList.push(subObjs)
							}
						})
						editList.map((items)=>{
							var objs={};
							for(var listVal in items){
								var listKey = vm.getNameByProp(listVal);
								objs[listKey] = items[listVal]
							}
							subList.push(objs)
						})
						params[key] = subList;
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
						var editData={
							value:field.fieldValue
						}
						params[key] = editData;
					};
				}
			});
			vm.$refs.ruleForm.validate(function(valid){
				if(valid) {
					var rowCode = vm.smallCellCode,
						url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
					$('#gsm_setting_main').addClass('loading');
					axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
							vm.closeSettings();
						}else{
							vm.$message.error(data["message"])
						}
						$('#gsm_setting_main').removeClass('loading');
					})
				}
			});
		},
		// 关闭设置页面
		closeSettings(){
			eventBus.$emit('gsm-close-setting');
		},
		getNameByProp(prop) {
			var vm = this,
				reg = /^\w*\.\d*\.\w*$/
				key = prop;
			
			if(reg.test(prop)) {
				var mReg = /\.(\d*)\./,
					sufReg = /\.(\w*)$/,
					idx = prop.match(mReg)[1],
					sufStr = prop.match(sufReg)[1];

				vm.codeList.map(function(name){
					var index = vm.indexs[name];
					if(vm.casts[name] == sufStr && index == idx) {
						key = name;
					}
				});
			}else {
				vm.codeList.map(function(name){
					if(vm.casts[name] == prop) {
						key = name;
					}
				});
			}

			return key;
		},
		createId(idVal,list){
			var vm = this,
				val = idVal + '';
			if(list.includes(val) == true){
				idVal += 1 ;
				return vm.createId(idVal,list);
			}else{
				return  idVal + '';
			}
		},
		//校验IP
        isValidIP(ip){
            var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
            return reg.test(ip);     
        },
        //Ipv6校验 
        isIPv6(str){ 
            var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
            return reg.test(str);
        },
        //校验子网掩码
        isMask(str){
            var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/; 
            return exp.test(str); 		
        },
        // 验证输入的是否是数字
        isNumeric(str) {
            if(str.length==0){
                return false;
            }
            for(var i=0;i<str.length;i++){
                if(str.charAt(i)<"0" || str.charAt(i)>"9"){
                    return false;
                }
            }
            return true;  
        },
		syncSettingsClick(){
			var vm = this,
				urls='${ctx}/cell/quicksettings/sync.action',
				params = {
					smallCellCode:vm.smallCellCode
				},
				str = Math.random().toString();
				
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				if(data["success"]){

				}else{
					vm.$message.error(data["message"])
				}
			})
		}
	},
	mounted() {
		eventBus.$off("gsm-data").$on("gsm-data",this.init)
	}
});

</script>
