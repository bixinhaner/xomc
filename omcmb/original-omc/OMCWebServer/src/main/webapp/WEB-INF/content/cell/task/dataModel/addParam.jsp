<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
    .split-line { border: none; border-top: 1px solid #e9e9e9; margin: 10px 0 20px 0; }
     #paramPage .el-select-mini input { min-height: 26px; }
    .basicInfoBox { margin: 16px 40px 10px; }
	.originalBox .el-checkbox { padding: 3px 6px 0 10px; }
	.addVersionWarp { width: 400px; height: 254px; border-radius: 4px; text-align: center; }
	.addVersionBtn { font-size: 20px; padding: 100px 0 10px; }
	.selectVersionBox .el-form-item__content { margin-left: 0px !important; }
	.selectVersionBox .el-form-item__content .el-input { width: 300px; }
	.licenseSwitchBox { position: absolute; top: 4px; }
	.licenseBox .el-query{ right: 0; }
	.licenseImportBox { margin: 0 10px; width: 26px; height: 26px; border: 1px solid #D7D7E6; text-align: center; border-radius: 8px; }
	.licenseImportBox i { font-size: 12px; line-height: 26px; }
	.leftWarp {flex: 1; height: 100%; position: relative; overflow: hidden; flex-direction: column; border-radius: 10px; }
	.titleBox { width: 100%; height: 38px; background: #FFFFFF; line-height: 38px; position: absolute; top: 0; left: 0; z-index: 100; }
	.footerBox { width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100; }
	.AddTitle { padding: 0 30px; }
	.leftClose { margin-right: 10px; }
	.closeIconBox .el-icon { font-size: 24px !important; }
	.specifyModifyRightBox { flex: 0 580px; height: 100%;  margin-left: 10px; border-radius: 10px; overflow: hidden; }
	.rightBox320 { flex: 0 320px; height: 100%;  margin-left: 10px; border-radius: 10px; overflow: hidden; }
	.rightBox { flex: 0 1 360px; height: 100%;  margin-left: 10px; border-radius: 10px; overflow: hidden; }
	.mainContent { width: 100%; overflow-y: scroll; height: 100%; position: absolute; top: 40px; z-index: 0; box-sizing: border-box; }
	.commonBackgroundWhite { background: #FFFFFF; }
	.commonTitle { height: 38px; line-height: 38px; }
	.rightHeaderBox .addTitle {  padding: 0; }
	.rightTextBox { padding: 10px 0; }
	.commonBorderBottom { border-bottom: 1px solid #E9EDF9; }
	.commonBorderTop { border-top: 1px solid #E9EDF9; }
	.originalVersionBox { padding: 16px 0 0; }
	.originalVersionBox .el-input{ width: 320px } 
	.width280 .el-input{ width: 280px; }
	.originalVersionBox .el-input__inner{ height: 30px; line-height: 30px; }
	.originalVersionBox .el-input-group__append { padding: 0 8px; background-color: #FFFFFF; }
	.originalVersionBox .el-input-group__append .el-icon { font-size: 14px; }
	.originalVersionBox .el-input-group__append .el-icon:before { color: #7A7992; }
	.originalVersionBox .el-table tr { display: none; }
	.ipErrorTip { color: #FA5555; font-size: 12px; }
	.versionResultBox { border-radius: 4px; background-color: #FFFFFF; margin-top: 5px; width: 318px; max-height: 122px; padding: 5px 0; overflow: auto;}
	.versionResultBox .el-form-item { margin-right: 0 !important;}
	.form-suffix { position: relative; margin: 3px 0 0 20px; padding: 0; width:278px; border: none; background: #fff; }
	.form-suffix:hover { background: #F4F9FF; border-radius: 100px; }
	.form-suffix .deleteVersion { display: none; position: absolute; right: 0; top: 8px; }
	.form-suffix:hover .deleteVersion { display: inline-block !important; }
	.form-suffix .text { padding-left: 10px; color: #666666; }
	.form-suffix .ipTextWarp { font-size: 14px; }
	.selectedVersion .queryGroup .el-input__inner { width: 220px !important; }
	.selectedVersion .el-query .advanceQuery { height: 28px; }
	.selectedVersion .el-query .advanceQuery .el-input.el-input--small { width: 230px !important; }
	.selectedVersion .el-query { right: 0px; }
	.operBtn { width: 26px; height: 26px; border: 1px solid #D7D7E6; border-radius: 8px; margin-top: 1px; }
	.operBtn .el-icon { line-height: 26px; }
	.operBtn .el-icon:before, .importIcon:before { color: #7A7992; }
	.container .group { padding-left: 26px; }
	.group-title { color: #7A7992; }
	.mainContent .el-form-item { margin-bottom: 22px; display: block}
	.mainContent .el-form-item__label { line-height: 28px; } 
	.enableCommon .el-switch { margin-top: 4px; }
	.selectCommon .el-select .el-input { width: 150px; }
	.inputCommon .el-input__inner { border-radius: 4px; }
    .paramTwoBox { padding: 10px 20px; }

	.commonText { padding: 3px 0; }
	.commonText1 { padding: 0 10px; }
	.el-collapse-item__content { padding-bottom: 0; }
	.enbIdItem .el-textarea__inner { height: 110px; resize: none; border-radius: 4px; width: 600px;}
	.reginDeployingBox .el-tabs--border-card { border: 1px solid #E9EDF9; box-shadow: none; -webkit-box-shadow: none; }
	.reginDeployingBox .el-tabs .el-tabs__header, .reginDeployingBox .el-tabs--border-card>.el-tabs__header { border-bottom: 1px solid #E9EDF9; }
	.reginDeployingBox .el-tabs--border-card>.el-tabs__header { background-color: #FFFFFF; }
	.reginDeployingBox .el-tabs__item { font-size: 14px; color: #7A7992; font-weight: normal;  }
	.reginDeployingBox .el-tabs--border-card>.el-tabs__header .el-tabs__item.is-active { color: #7A7992; font-weight: bold;  border-right-color: #E9EDF9; border-left-color: #E9EDF9; }
	.reginDeployingBox .el-tabs--border-card>.el-tabs__header .el-tabs__item:not(.is-disabled):hover { color: #4D84FF; }
	.modifyForm .el-form-item { margin-bottom: 20px; display: inline-block; margin-right: 60px; }
	.modifyForm .el-form-item .el-form-item__label { margin-bottom: 4px; }
	.commonTabletitle span { flex: 1; }
	.resultVersionTable thead { display: none; }
	.resultVersionTable tbody tr td { border: none; background-color: #FFFFFF; }
	.table-suffix { border: none; position: relative; }
	.table-suffix .deleteVersion { display: none; position: absolute; right: 0; top: 6px; }
	.table-suffix:hover .deleteVersion { display: inline-block !important; }
	.table-suffix .text { padding-left: 10px; color: #666666; }
	.table-suffix .ipTextWarp { font-size: 14px; }

	.paramPoolWarp .el-form-item { width: 49%; display: inline-block; flex-direction: row; }
	.paramPoolWarp .el-form-item .el-input { width: 186px; }
	.commonLeft10 { margin-left: 2px; }
	.fielset-cls {margin-top: 15px; padding: 0px 20px; border: none; border-top: 1px solid #e9e9e9; height: 0px; overflow: hidden;}
	.fielset-cls.extended {height: auto; padding: 10px 20px; border-radius: 5px; border: 1px solid #e9e9e9;}
	.fielset-cls .el-form-item { max-width: 360px;}
	.commonFormFotter { width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100; }
	.el-form-item__error { padding-top: 0 !important; }
	.dialogIpsec { margin: 50px auto 0 !important;}
	.el-input.is-disabled .el-input__inner { height: 26px !important; }
	.container .slide-position-top .el-icon-close { font-size: 14px; }
	.marginRight10 { margin-right: 10px; }
	.el-radio-button__inner { padding: 6px 20px; font-size: 12px; border-left: 1px solid #DCDFE6; }
	.el-radio-button__orig-radio:checked+.el-radio-button__inner { background-color: #f2f6ff;  color: #7a7992; }
	.commonBorder2 { border: 1px solid #E9EDF9; }
	#paramPage .el-input__suffix { margin-top: 0px; }
	.selectPublicPath{
		padding:0 10px;
		min-width:160px;
		height:24px;
		line-height:21px;
		background:#F2F6FF;
		border:1px solid #4D84FF;
		border-radius:4px;
		display:inline-block;
		text-align:center;
		color:#333333;
	}
	.publicPathDelete{
		font-size:12px;
		margin-left:10px;
		line-height:24px;
		position: unset !important;
	}
</style>

<div id="paramPage" style='background: #F6F7FB; position: relative;'>
   <div class='commonFlex commonContent' style='height: 99.8%; width:100%;'>
   		<!-- 左侧主体内容 -->
   		<div class='leftWarp commonBorder2 commonBackgroundWhite'>
   			<div class='commonFlex commonContent titleBox commonTitle commonBorderBottom'>
   				<span class='AddTitle commonColor commonFontSize14 commonFontWeight'>{{headerTitle}}</span>
   				<div class="circleIcon placeholder-bt addBtnStyle" placeholder="<%=rb.getString("GuanBi")%>" style='top: 8px;'>		
					<span class="el-icon-circle-close el-icon" @click="addOrUpdateCancel"></span>
				</div>
   			</div>
   			<div class='mainContent' style='bottom: 70px; height: auto;'>
   				<el-form ref="addOrEditForm" :model="addOrEditForm" :rules="addOrEditRule" label-position="top" label-width="125" >			
                    <div class="basicInfoBox commonColor">
						<!--  basic info --> 
						<div>
							<el-form-item label="<%=rb.getString("GongYouPath")%>" prop="standardPath" class='inputCommon' v-if='curType != "add"'>								
				                <el-input v-model="addOrEditForm.standardPath" disabled style="width: 300px;"></el-input>
				            </el-form-item>
				            <el-form-item label="" prop="standardPath" style='margin-bottom: 18px;' v-if='curType == "add"'>	
								<el-ctable ref="selectProductModelTable" :url="pathUrl" class="commonBorder2" :disabled="isReadOnly"
					            	:row-key="'standardPath'" :query-params="publicPathParam"
					            	@row-click='productModelBatchSelect' height="210px">	                          
					                <div slot="toolbar">
										<div class='commonFlex commonToolBarBox' style='position: relative;'>
											<span class='commonColor commonFontSize14' style='margin: 0 20px;'><%=rb.getString("GongYouPath")%></span>
											<div style="position:absolute;top: 6px;left:150px;" v-show="addOrEditForm.standardPath != ''">
												<span class="selectPublicPath" >{{addOrEditForm.standardPath}}<i class="el-icon el-icon-close publicPathDelete" @click="selectPublicPathDelete"></i></span>
											</div>
											<div class='commonToolBarBox'>
												<el-query type="normal" @query="queryModelName" placeholder="<%=rb.getString("GongYouPath") %>"></el-query>
											</div>
										</div>
									</div>
					                <el-table-column width="50">
										<div slot-scope="scope" style="margin: 0 auto;">
											<el-radio v-model="addOrEditForm.standardPath" :label="scope.row.standardPath"><span></span></el-radio>
										</div>
									</el-table-column>
									<el-table-column label="<%=rb.getString("GongYouPath") %>" prop="standardPath"></el-table-column> 	                          
					            </el-ctable>   
							</el-form-item>
							<div class='commonFlex'>
								<el-form-item label="<%=rb.getString("SiYouPath")%>" prop="privatePath" class='inputCommon' style='margin-right: 100px; '>
					                <el-input v-model="addOrEditForm.privatePath" :disabled="curType == 'readonly'" style="width: 300px;"></el-input>
					            </el-form-item>
					            <el-form-item label="<%=rb.getString("DuXieQuanXian")%>" prop="writable" placeholder="<%=rb.getString("QingXuanZe") %>" class='selectCommon'>
					                <el-select v-model="addOrEditForm.writable" :disabled="curType != 'add'">
					                    <el-option v-for="item in productList" :label="item.name" :value="item.value"></el-option>
					                </el-select>
					            </el-form-item>
				            </div>
				            <div class='commonFlex'>
				            	<el-form-item label="<%=rb.getString("ShuJuLeiXing")%>" prop="dataType" placeholder="<%=rb.getString("QingXuanZe") %>" class='selectCommon' style='display: inline-block;'>
					                <el-select v-model="addOrEditForm.dataType" :disabled="curType != 'add'">
					                    <el-option v-for="item in dataTypeList" :label="item.name" :value="item.value"></el-option>
					                </el-select>
					            </el-form-item>
					            <el-form-item prop="rangeType" label=" " style='display: inline-block; margin-left: 10px; margin-top: 28px;' v-show='addOrEditForm.dataType != "boolean" && addOrEditForm.dataType != "dateTime"'>
						            <el-radio-group v-model='addOrEditForm.rangeType' :disabled="curType != 'add'">
										<el-radio-button label="num"><%=rb.getString("ChangDuFanWei")%></el-radio-button>
										<el-radio-button label="enum"><%=rb.getString("MeiJuFanWei")%></el-radio-button>	
									</el-radio-group>
								</el-form-item>
				            </div>
				            <!--  长度 是按照 string :maxlength minlength init: max min-->
				            <el-form-item label="<%=rb.getString("QuZhiFanWeiBanMian")%>" prop="commonMaxMinLength" v-show='addOrEditForm.dataType != "boolean" && addOrEditForm.dataType != "dateTime" && addOrEditForm.rangeType == "num"'>
				                 <el-input v-model="addOrEditForm.commonMaxMinLength" :disabled="curType != 'add'" style="width: 230px;" ></el-input>
				                 <span class='commonColor2 commonFontSize12' style='display: block; line-height: 18px;'><%=rb.getString("ZuiXiaoZhi")%>：<%=rb.getString("ZuiDaZhi")%> <%=rb.getString("LiRu")%>： '30:33'</span>
				            </el-form-item>
				            <!--  枚举-->
				            <div class='commonFlex' v-show='addOrEditForm.dataType != "boolean" && addOrEditForm.dataType != "dateTime" && addOrEditForm.rangeType =="enum"'>
				            	<el-form-item label="<%=rb.getString("QuZhiFanWeiBanMian")%>" prop="enumValueOfUser" style='display: inline-block;'>
					                 <el-input v-model="addOrEditForm.enumValueOfUser" :disabled="curType != 'add'" style="width: 230px;"></el-input>
					                 <span class='commonColor2 commonFontSize12' style='display: block; line-height: 18px;'><%=rb.getString("CanShuZhi")%> <%=rb.getString("LiRu")%>： '100ms:100ms'</span>
					            </el-form-item>
					            <el-form-item label="<%=rb.getString("QuZhiFanWeiMingLing")%>" prop="enumValueOfDevice" style='display: inline-block; margin-left: 128px;'>
					                 <el-input v-model="addOrEditForm.enumValueOfDevice" :disabled="curType != 'add'" style="width: 230px;" ></el-input>
					                 <span class='commonColor2 commonFontSize12' style='display: block; line-height: 18px;'><%=rb.getString("CanShuZhi")%> <%=rb.getString("LiRu")%>： '100:100'</span>
					            </el-form-item>
				            </div>
				             <p class='ipErrorTip' style='margin-top: -6px;'>{{numEnumError}}</p>
				            <el-form-item label="<%=rb.getString("DongTaiShengXiao")%>" prop="isDynamic" placeholder="<%=rb.getString("QingXuanZe") %>" class='selectCommon basicLeftLabel'>
				                <el-select v-model="addOrEditForm.isDynamic" :disabled="curType != 'add'">
				                    <el-option v-for="item in isDynamicList" :label="item.name" :value="item.value"></el-option>
				                </el-select>
				            </el-form-item>
				            <el-form-item label="<%=rb.getString("EnbBeiZhu")%>" prop="privateName" class='enbIdItem'>
	                            <el-input type="textarea" v-model="addOrEditForm.privateName" :rows="3" :disabled="curType != 'add'"></el-input>
	                        </el-form-item>
						</div>
					</div>					
                </el-form>   				
   			</div>
		    <div class='commonFlex commonContent footerBox commonBorderTop' v-show="curType !='readonly'">
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" size="mini" @click='addOrUpdateSubmit' :disabled='saveBtnDisabled'><%=rb.getString("QueDing")%></el-button>
					<el-button size="mini" @click='addOrUpdateCancel'><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>            
   </div>
</div>

<script>
    var addOrEditConfig = new Vue({
        el: '#paramPage',
        data() {
            var vm = this,
         
           		validatePrivatePath = function(rule,value,callback){			  
					if(vm.curType == 'readonly'){
						callback();
					}else{
						if(value === '' || value === null || value === undefined) {
							callback(new Error('<%=rb.getString("BiTian")%>'));
						}else{
							callback();
						}
					}	
				},
				// 详情、 数据类型-boolean、数据类型-dateTime 时，都不校验
				validateCommonValue = function(rule,value,callback){			  
					if(vm.curType == 'readonly' || vm.addOrEditForm.rangeType != 'enum' || vm.addOrEditForm.dataType == 'boolean' || vm.addOrEditForm.dataType == 'dateTime'){
						callback();
					}else{
						if(value === '' || value === null || value === undefined) {
							callback(new Error('<%=rb.getString("BiTian")%>'));
						}else{
							callback();
						}
					}
				},
				validateCommonLength = function(rule,value,callback){			  
					var reg = /^-?[0-9:]+.?[0-9:]*/;
					var curValue = value.indexOf(':') != -1;
					var curText = value.split(':');
					vm.curStartValue = Number(value.split(':')[0]);
					vm.curEndValue = Number(value.split(':')[1]);

					if(vm.curType == 'readonly' || vm.addOrEditForm.rangeType != 'num' || vm.addOrEditForm.dataType == 'boolean' || vm.addOrEditForm.dataType == 'dateTime'){
						callback();
					}else{
						if(value === '' || value === null || value === undefined || curValue == false || value.split(':')[0] === '' || value.split(':')[1] === '' || vm.curStartValue === NaN || vm.curEndValue === NaN){
							callback(new Error('%=rb.getString("BiTian")%>'));
						}else if(!reg.test(value) || vm.curStartValue >= vm.curEndValue){
							callback(new Error("format error"));
						}else{
							callback();
						}
					}
				}
				
            return {
            	
				addOrEditForm: {
					standardPath: '',
            		privatePath: '',
            		writable: 'R',
            		dataType: 'int',
            		rangeType: 'num',
            		commonMaxMinLength: '',
            		enumValueOfUser: '',
            		enumValueOfDevice: '',
            		isDynamic: '0',
            		
            		privateName: '',
            		minLength:'',
            		maxLength:'',
            		min:'',
            		max: ''
				},
				addOrEditRule: {
					//standardPath: [{validator: validatePrivatePath}],
					privatePath: [{validator: validatePrivatePath}],
					//commonMaxMinLength: [{validator: validateCommonLength}],//最小值 最大值，最小，最多长度
					//enumValueOfUser: [{validator: validateCommonValue}],
					//enumValueOfDevice: [{validator: validateCommonValue}],
				},
				publicPathParam: {
					search_text: '',
					timeZone: timeZone,
					page: 1,
					rows: 50
				},
				headerTitle: '',
				isReadOnly: false,
				numEnumError: '',
				curType: '',
				productList: [
					{ name: '<%=rb.getString("ZhiDu")%>', value: 'R' },
					{ name: '<%=rb.getString("ZhiXie")%>', value: 'W' },
					{ name: '<%=rb.getString("DuXie")%>', value: 'RW' }
				],
				dataTypeList: [
					{ name: 'Boolean', value: 'boolean' },
					{ name: 'DateTime', value: 'dateTime' },
					{ name: 'Int', value: 'int' },
					{ name: 'UnsignedInt', value: 'unsignedInt' },
					{ name: 'String', value: 'string' }
				],
				isDynamicList: [
					{ name: '<%=rb.getString("JingTaiShengXiao")%>', value: '0' },
					{ name: '<%=rb.getString("DongTaiShengXiao")%>', value: '1' }
				],
				saveBtnDisabled: false,
				curRowDataId: '',
				curParamModel: '',
				curStartValue: '',
				curEndValue: '',
				
            	selectProductModelList: [],
            	addProductModelShow: false,
            	caFileId:'',
            	pathUrl: '${ctx}/dataModel/enb/standardPrivateRela/getStandardPathList.action'
            }
        },
        computed: {
			
        },
        watch: {
            'addOrEditForm.enumValueOfUser':function(newVal,oldVal){
    			var vm = this;
				if(newVal === '' || newVal === null || newVal === undefined ){
					vm.numEnumError = '';
				}else{
					var enumLength = vm.addOrEditForm.enumValueOfDevice.split(',').length;
					if(newVal.split(',').length == enumLength){
						vm.numEnumError = '';
					}else{
						vm.numEnumError = '<%=rb.getString("QuZhiFanWeiShuRuTiaoJian")%>';
					}
				}
			},
			'addOrEditForm.enumValueOfDevice':function(newVal){
    			var vm = this;
				if(newVal === '' || newVal === null || newVal === undefined){
					vm.numEnumError = '';
				}else{
					var numLength = vm.addOrEditForm.enumValueOfUser.split(',').length;
					if(newVal.split(',').length == numLength){
						vm.numEnumError = '';
					}else{
						vm.numEnumError = '<%=rb.getString("QuZhiFanWeiShuRuTiaoJian")%>';
					}
				}
			},
        },
        methods: {
        	queryModelName(text) {
				this.publicPathParam.search_text = text;
			},
			selectPublicPathDelete(){
				var vm = this;	
				vm.addOrEditForm.standardPath = '';
				vm.$refs.selectProductModelTable.clearSelection();
			},
            init(type, parmaMode, row) {
                var vm = this;
                vm.curRowDataId = row.id;
                vm.curType = type;
                vm.curParamModel = parmaMode;
               	//row  当前行数据，详情和修改
				vm.isReadOnly = type == 'readonly';
                if(type == 'add'){
                	vm.headerTitle = '<%=rb.getString("TianJiaCanShu")%>' + '(' + parmaMode + ')';
                	vm.curType = 'add'
                }else if(type == 'modify'){
                	vm.headerTitle = '<%=rb.getString("XiuGaiCanShu")%>' + '(' + parmaMode + ')';
                	vm.curType = 'modify';
                	vm.commonInit(row);
                }else{
                	vm.headerTitle = '<%=rb.getString("ChaKanCanShu")%>' + '(' + parmaMode + ')';
                	vm.curType = 'readonly';
                	vm.commonInit(row);
                }
            },
            commonInit(rowData){
            	var vm = this,
         		paramList = ['standardPath','privatePath','writable','dataType','rangeType','enumValueOfUser','enumValueOfDevice','isDynamic','privateName'];
            	paramList.map(function(prop){
         			vm.addOrEditForm[prop]= rowData[prop];						
 				});
				var curLength = rowData.minLength + ":" + rowData.maxLength,
					curNum =  rowData.min + ":" + rowData.max;
				
            	// 数据类型： string 取值是 minLength, maxLength;  数据类型： init 取值 max, min
            	if(vm.addOrEditForm.dataType == 'string' && vm.addOrEditForm.rangeType == 'num'){
					vm.addOrEditForm.commonMaxMinLength = curLength;
				}else if((vm.addOrEditForm.dataType == 'int' || vm.addOrEditForm.dataType == 'unsignedInt') && vm.addOrEditForm.rangeType == 'num'){
					vm.addOrEditForm.commonMaxMinLength = curNum;
				}else{
					vm.addOrEditForm.commonMaxMinLength = '';
				}

            	setTimeout(function(){
					initForm(vm.$refs.addOrEditForm);
				},100)
            },
            productModelBatchSelect(row, old){
    			var vm = this;
    			if(row){
    				vm.addOrEditForm.standardPath = row.standardPath;
    				vm.$refs['addOrEditForm'].clearValidate('standardPath');
    			}
    		},

            addOrUpdateSubmit(){
				var vm = this, params = {}, url = '';  
        		params.platformType = vm.curParamModel;// 当前选中的param-model
				params.standardPath = vm.addOrEditForm.standardPath; //公有path 
				params.privatePath = vm.addOrEditForm.privatePath;//私有path 
				params.writable = vm.addOrEditForm.writable;//读写权限
				params.dataType = vm.addOrEditForm.dataType;//数据类型
				
				params.enumValueOfUser = vm.addOrEditForm.enumValueOfUser;//枚举范围中的版面
				params.enumValueOfDevice = vm.addOrEditForm.enumValueOfDevice;//枚举范围中的命令下发
				params.isDynamic = vm.addOrEditForm.isDynamic; // 1-动态； 0-静态
	
				params.privateName = vm.addOrEditForm.privateName; // 备注
				params.paramModel = vm.curParamModel; // param_model
				params.rangeType = vm.addOrEditForm.rangeType; // num-长度范围； enum - 枚举范围
				if(vm.addOrEditForm.dataType == 'string' && vm.addOrEditForm.rangeType == 'num'){
					params.minLength = vm.curStartValue;// 长度范围 rangeType    并且是是string 
					params.maxLength = vm.curEndValue;// 长度范围
					params.min = '';
					params.max = '';
				}else if((vm.addOrEditForm.dataType == 'int' || vm.addOrEditForm.dataType == 'unsignedInt') && vm.addOrEditForm.rangeType == 'num'){
					params.min = vm.curStartValue;// 长度范围 rangeType    如果是int，unsignedInt
					params.max = vm.curEndValue;// 长度范围
					params.minLength = '';
					params.maxLength = '';
				}else{
					params.minLength = '';
					params.maxLength = '';
					params.min = '';
					params.max = '';
				}

            	if(vm.curType == 'modify'){
            		params.id = vm.curRowDataId;
            		url = "${ctx}/dataModel/enb/standardPrivateRela/modifyInfo.action";
            	}else{
            		url = "${ctx}/dataModel/enb/standardPrivateRela/addInfo.action";
            	}
            	var saveParams = JSON.stringify(params);
            	vm.$refs.addOrEditForm.validate(function(valid){
            		 if(valid && vm.numEnumError == ''){
 						vm.saveBtnDisabled = true;
                         axios.post(url, saveParams, {headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(res){
                         	var data = res.data;
                             if(data['success'] == true) {
                                 vm.$message({
 		    						message: '<%=rb.getString("ChengGong")%>',
 		    						type:'success',
 		    					});
                                eventBus.$emit('reload-config-list');
                                vm.addOrUpdateCancel();
                             }else {
                                 vm.$message.error(data['message']);
                             }
                             vm.saveBtnDisabled = false;
                         }).catch(function(){});
 					}else{
 					}
 				})
            },
            addOrUpdateCancel(){
            	var vm = this;
				eventBus.$emit('cancel-slide')
            },
        },
        mounted() {
        	var vm = this;
            eventBus.$off('save-config').$on('save-config', vm.addOrUpdateSubmit);
            eventBus.$off('hander-cancel').$on('hander-cancel',vm.addOrUpdateCancel);
            eventBus.$off('init-config').$on('init-config', vm.init);  
        }
    });
</script>

