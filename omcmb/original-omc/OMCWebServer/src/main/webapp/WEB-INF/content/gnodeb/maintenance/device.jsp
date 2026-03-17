<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<!DOCTYPE html>
<html>
<head>
<title>Device</title>
<style>
	.selected-title {
		position: relative;
		display: flex; 
		justify-content: space-between;
		padding: 10px;
	}
	.selected-title::after {
		content: '';
		display: block;
		position: absolute;
		height: 1px;
		left: -12px;
		right: -12px;
		bottom: 0px;
		background: #e9e9e9;
	}
  .no-padding .el-dialog__body {
    padding: 0px !important;
  }
  .fixed-height .el-dialog__body {
    max-height: 80%;
    display: flex;
    flex-direction: column;
  }
  #gnodeb_device .treeItemBoxCls{
      width: 300px;
      position: relative;
	}
	#gnodeb_device .treeItemBoxCls .operCls{
      position:absolute;
      right:3px;
      z-index: 66;
	}
  #gnodeb_device  .groupMgmt{
      width: 300px;
  }
  #gnodeb_device  .groupMgmt .el-input.el-input--small{
		width: 200px;
	}
	#gnodeb_device  .groupMgmt .el-tree-node__content{
		height: 30px;
	}
  #gnodeb_device .groupTreeBox{
		height: calc(100% - 44px);
		overflow: auto;
	}
	#gnodeb_device .ItemLabelCls{
		display: inline-block;
		width: 220px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>
</head>
<body>
  <div id="gnodeb_device" class="container" style="background: #fff;">
    <div class="rend-in-two">
      <!-- 设备组列表 -->
      <div label="<%=rb.getString("SheBeiZu")%>" class="groupMgmt">
        <!-- 操作按钮 -->
        <div class="operations">
          <div v-if="isAdmin" class="placeholder-bt" tip="<%=rb.getString("XinZeng")%>" @click="addDeviceGroup">
            <span class="el-icon el-icon-circle-add"></span>
          </div>
        </div>
        
        <el-query type="normal" @query="queryGroupList" placeholder="<%=rb.getString("SheBeiZuMingCheng")%>" style="margin-bottom:10px;"></el-query>
        <div class="groupTreeBox">
          <el-tree 
            ref="groupTree"
            :data="groupData"
            node-key="id"
            :default-expanded-keys="defaultexpandedKeys"
            :default-checked-keys="defaultCheckedKeys"
            :props= "{label:'group_name'}"
            :highlight-current="true"
            @node-click="groupRowClick"
          >
            <div class="treeItemBoxCls" slot-scope="{ node,data }">
              <span class="ItemLabelCls" :title="node.label">{{node.label}}</span>
              <span v-if="isWritable" class="el-icon el-icon-operation-more operCls" @click="groupOpClick(node,data,event)" v-clickoutside="handerClose"></span>
            </div>
          </el-tree>
        </div>
      </div>

      <!-- 设备列表 -->
      <div>
        <!-- 操作按钮 -->
        <div class="operations">
          <div class="placeholder-bt CODE_GNB hidden" placeholder="<%=rb.getString("XinZeng")%>" @click="addGnb">
            <span class="el-icon el-icon-circle-add"></span>
          </div>
          <el-popover placement="bottom-start">
            <div slot="reference" class="placeholder-bt CODE_GNB hidden" placeholder="<%=rb.getString("DaoRu")%>">
              <span class="el-icon el-icon-circle-import"></span>
            </div>
            <div class="selected-title">
              <span style="font-size: 14px;font-weight: bold;"><%=rb.getString("WenJianDaoRu")%></span>
              <span class="el-icon el-icon-close" @click="closeImport"></span>
            </div>
            <div style="padding:10px;">
               <!-- 文件表单 -->
              <el-form>
                <el-form-item label="File Name" style="display: flex;align-items: center;margin-top: 20px;">
                  <el-upload name="uploadFile" ref="upload" 
                    :on-success='checkFile' 
                    :on-change="fileChange"  
                    :show-file-list="false"
                    :action="uploadFileURL" 
                    :data="fileParams"
                    :auto-upload="false">
                    <el-input :readonly="true" :value="fileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
                      <a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect" style="margin-top: 4px;"></a>
                    </el-input>
                    <div slot="tip" class="el-upload__tip" v-show="!typeFlag"><%=rb.getString("DaoRuWenJianGeShi")%></div>
                    <div slot="tip" class="el-upload__tip" v-show="selectFlag"><%=rb.getString("QingXianXuanZeWenJian")%></div>
                    <a slot="trigger" ref="file_up"></a>
                  </el-upload>
                </el-form-item>
              </el-form>
              <!-- 下载提示 -->
              <p style='color:#999;margin-top:10px;white-space: nowrap;'>
                <span class='el-icon el-icon-circle-info' style='font-size:14px;margin-right:5px;'></span>
                Please import a file as the format specified in the sample tempalte.
                <span class='curpo' @click="exportTemplate">
                  <span style='vertical-align:top' class='el-icon el-icon-common-download'></span>
                  <span style='color:#363B4E;text-decoration:underline'>Export Template</span>
                </span>
              </p>
              <!-- 按钮 -->
              <el-button-group size="mini" style='margin-top:30px;'>
                <el-button type="primary" size="mini" @click="uploadDevice">OK</el-button>
                <el-button size="mini" @click="closeImport">Cancel</el-button>
              </el-button-group>
            </div>
             
          </el-popover>
          <div class="placeholder-bt" placeholder="<%=rb.getString("DaoChu")%>" @click="exportDevice">
            <span class="el-icon el-icon-circle-export"></span>
          </div>
        </div>

        <!-- 查询区域 -->
        <div class="toolbar" label="gNB">
          <el-query type="normal" @query="queryDevices" placeholder="<%=rb.getString("XiaoZhanBianMa")%>"></el-query>
        </div>

        <!-- 设备列表 -->
        <el-ctable id="devices" ref="ctableDevice" row-key="serial_number"
          :url="deviceUrl"
          :query-params="params_device"
          @selection-change="change">
          <el-table-column v-if="isWritable && groupWritable" label='' width="50" :reserve-selection="true" type="selection" prop="ck"></el-table-column>
          <el-table-column v-if="isWritable" label='' width="30" prop="">
            <template slot-scope="scope">
              <div class="el-icon el-icon-operation-more" @click="optDeviceClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
            </template>
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
          <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="100" prop="serial_number"></el-table-column>
          <el-table-column label='<%=rb.getString("MACDiZhi")%>' min-width="100" prop="mac_address"></el-table-column>
        </el-ctable>
        
		    <el-cmenu ref="menuDevices" :data="menusDevices" @click="clickDeviceMenu"></el-cmenu>
      </div>
    </div>

    <!-- 批量操作浮层 -->
    <el-bulk ref="bulk" target="devices" :list="ckRows" row-key="serial_number"
      :message="{title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("XiaoZhanBianMa")%>',clear:'<%=rb.getString("QingChu")%>',cancel: '<%=rb.getString("QuXiao")%>'}">
      <template slot="button">
        <a class="linkbutton linkbutton_nowanna" v-show="groupWritable" @click="deleteCells"><span><%=rb.getString("PiLiangShanChu")%></span></a>
        <a class="linkbutton linkbutton_trend" @click="movecells"><span><%=rb.getString("YiDongDaoSheBeiZu")%></span></a>
      </template>
    </el-bulk>

    <!-- 设备组菜单 -->
		<el-cmenu ref="menuGroup" :data="menusGroup" @click="clickMenu"></el-cmenu>

    <!--添加新一级设备组-->
    <el-dialog 
      :title='dialogTitle' 
      :visible.sync="showDeviceGroupDialog" 
      ref="windowDialog" 
      :width="deviceGroupDialogWidth"  
      :height='deviceGroupDialogHeight'
      :close-on-click-modal="false"  
      @close='closeDeviceGroupDialog'  
    >
      <el-form  
        :model="groupForm"
        ref="deviceGroupDialogForm" 
        :rules="groupRules" 
        label-width="120" 
        label-position="left" 
        id="deviceGroupDialogForm" 
        :hide-required-asterisk='true'
      >
        <el-form-item label='<%=rb.getString("SheBeiZuMingCheng")%>' prop='groupName'>
          <el-input :disabled="viewDeviceGroupDialog" v-model="groupForm.groupName" style='width:200px;padding-top:7px;'></el-input>
        </el-form-item>
        <el-form-item label='<%=rb.getString("MiaoShu")%>' prop='description' v-if="false">
          <el-input type="textarea" maxlength=50  style='width:440px;' :disabled="viewDeviceGroupDialog"></el-input>
        </el-form-item>
      </el-form>
      <span slot="footer" v-show="!viewDeviceGroupDialog">
        <div>
          <el-button type="primary" @click="addDeviceGroupSubmit"><%=rb.getString("QueDing")%></el-button>
          <el-button @click="closeDeviceGroupDialog"><%=rb.getString("QuXiao")%></el-button>
        </div>
      </span>
    </el-dialog>
    <!-- 设备组 Dialog -->
    <el-dialog :custom-class="'fixed-height'"
    	:title='dialogTitle' :visible.sync="showWindowInfo" ref="windowDialog" :width="windowWidth" top="10vh" :custom-class="dialogCls"
    	:close-on-click-modal="false" :url="dialogUrl" @close='closeDialog' @success="openDialogSuc">
    </el-dialog>

    <!-- 设备修改 Dialog -->
    <el-dialog title='<%=rb.getString("XiuGai")%>' :visible.sync="showModifyDevice" ref="modifyDialog" :width="windowWidth" 
      :close-on-click-modal="false"  @close='showModifyDevice = false' >
      <el-form ref="modifyForm" label-position="top" :model="deviceForm" :rules="deviceRules">
        <el-form-item>
          <%=rb.getString("XiaoZhanBianMa")%><%=rb.getString("MaoHao")%> {{deviceName}}
        </el-form-item>
        <el-form-item label="<%=rb.getString("JingDu")%>" prop="longitude">
          <el-input v-model="deviceForm.longitude"></el-input>
        </el-form-item>
        <el-form-item label="<%=rb.getString("WeiDu")%>" prop="latitude">
          <el-input v-model="deviceForm.latitude"></el-input>
        </el-form-item>
        <el-form-item label="<%=rb.getString("GaoDu")%>" prop="height">
          <el-input v-model="deviceForm.height"></el-input>
        </el-form-item>
      </el-form>
      <div>
        <el-button type="primary" @click="saveModifyDevice"><%=rb.getString("QueDing")%></el-button>
        <el-button @click="showModifyDevice = false"><%=rb.getString("QuXiao")%></el-button>
      </div>
    </el-dialog>

    <!-- 设备移动 Dialog -->
    <el-dialog ref="moveDialog" title="<%=rb.getString("YiDongDaoSheBeiZu")%>" :visible.sync="moveDlShow" width="600px">
      <el-ctable ref="moveTable" :url="moveGroupUrl" height="400px" row-key="id">
        <el-table-column label='' width="35" prop="">
          <template slot-scope="scope">
            <el-radio v-model="groupId" :label="scope.row.id">&nbsp;</el-radio>
          </template>
        </el-table-column>
        <el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' min-width="150" prop="group_name"></el-table-column>
      </el-ctable>
      <div>
        <el-button type="primary" @click="moveDevicesSend"><%=rb.getString("QueDing")%></el-button>
        <el-button @click="closeMoveDl"><%=rb.getString("QuXiao")%></el-button>
      </div>
    </el-dialog>

    <!-- 删除设备组弹窗 -->
    <el-dialog title="<%=rb.getString("QueRen")%>" :visible.sync="showDeviceGroupInfo" width="400" 
      :close-on-click-modal="false" @close="showDeviceGroupInfo = false">
      <div><%=rb.getString("QueDingShanChuSheBeiZu")%></div>
      <div v-show="delGroupType == 'stair'" style="font-size:12px;color:#999999;padding-top:8px;"><%=rb.getString("ZuNeiZiJiSheBeiZuJiSheBeiHuiBeiZiDongYiZhi")%></div>
      <div v-show="delGroupType == 'second'" style="font-size:12px;color:#999999;padding-top:8px;"><%=rb.getString("ZuNeiSheBeiHuiBeiZiDongYiDongDaoMoRenFenZu")%></div>
      <span slot="footer" class="dialog-footer">
        <div class="buttonGroup">
          <el-button type="primary" @click="deleteDeviceGroup"><%=rb.getString("QueDing")%></el-button>
          <el-button @click="showDeviceGroupInfo = false"><%=rb.getString("QuXiao")%></el-button>
        </div>	
      </span>
    </el-dialog>
  </div>

  <script>
   var gnbDeviceVue = new Vue({
      el: '#gnodeb_device',
      data() {
        var vm = this,
            /**
            * 验证输入的字符长度
            * @param str{string}   value值
            * @param num{number}   长度
            */ 
            lessThenLength = function(str,num){
                str += '';
                return str.replace(/\./g,'').length <= num;
            },
            validateGroupName = (rule,value,callback) => {
              var reg = /^[a-zA-Z0-9_\u4e00-\u9fa5,\s]{1,50}$/;
              
              if(value){
                if(reg.test(value)){
                  callback()
                }else{
                  callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>'))
                }
              }else{
                callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXianHanZi")%>'))
              }
            },
            // 经度 表单验证规则
            longitudeValidator = function(rule,value,cb) {
              if(value) {
                if(isNaN(value)) {
                    cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
                  }else if(value<-180 || value>180){
                    cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
                  }else {
                    var arr = value.split('.'),
                      precision = arr[1]||'';
                    if(precision.length>6) {
                      cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
                    }else cb();
                  }
                }else {
                  cb();
                }
            },
            // 维度 表单验证规则
            latitudeValidator = function(rule,value,cb) {
              if(value) {
                if(isNaN(value)) {
                  cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
                }else if(value<-90 || value>90){
                  cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
                }else {
                  var arr = value.split('.'),
                    precision = arr[1]||'';
                  if(precision.length>6) {
                    cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
                  }else cb();
                }
              }else {
                cb();
              }
            },
            // 高度 表单验证规则
            heightValidator = function(rule,value,cb) {
              if(value) {
                if(value<=99999999 && value-0>=0) {
                  cb();
                }else {
                  cb('<%=rb.getString("QuZhiFanWei")%>: 0-99999999');
                }
              }else {
                cb();
              }
            },
            // 距离 表单验证规则
            distanceValidator = function(rule,value,cb) {
              if(value) {
                if(value<=99999999 && value-0>=0) {
                  cb();
                }else {
                  cb('<%=rb.getString("QuZhiFanWei")%>: 0-99999999');
                }
              }else {
                cb();
              }
            };

        return {
          ckRows: [],
          menusGroup: [],
          menusDevices: [],
          dialogUrl: '',
          dialogTitle: '',
          showWindowInfo: false,
          windowWidth: '500px',
          moveType: '',
          rowDataGroup: '',
          rowDataDevice: '',
          fileParams: {
            FileName: '',
            group_id: '',
            isGnb: 1,
          },
          selectFlag:false,        //标识是否选择了文件 
          typeFlag:true,           //校验已选择的文件格式 
          fileName: '',
          params_device: {
            isGnb: 1,
            group_id:'',
            search_text:'',
            like_fields:'serial_number,macaddress'
          },
          uploadFileURL: '${ctx}/system/deviceGroup/uploadFile.action',
          deviceUrl: '',
          deviceForm: {
            cell_code: '',
            longitude: '',
            latitude: '',
            height: '',
            distance: '',
            isGnb: 1
          },
          deviceRules: {
            longitude: [
              {validator: longitudeValidator}
            ],
            latitude: [
              {validator: latitudeValidator}
            ],
            height: [
              {validator: heightValidator}
            ],
            distance: [
              {validator: distanceValidator}
            ]
          },
          showModifyDevice: false,
          deviceName: '',
          curGroup: '',
          moveDlShow: false,
          moveGroupUrl: '',
          groupId: '',
          groupForm:{
            groupName:'',
          },
          groupRules:{
            groupName:[
              {validator:validateGroupName,trigger:'blur'}
            ],
            description:[
              {max:100,trigger:'blur'}
            ]
          },
          showDeviceGroupDialog:false,
          viewDeviceGroupDialog:false,
          deviceGroupDialogHeight:'600px',
			    deviceGroupDialogWidth:'500px',
          stairGroupType:'',
          groupData:[
            // {id:1,group_name:'Default Level Group',children:[{id:5,group_name:'Default Device Group',built_in:'1'}],built_in:'1'},
            // {id:2,group_name:'ENB Group',children:[{id:6,group_name:'ENB1',built_in:'0'}],built_in:'0'}
          ],
          showDeviceGroupInfo:false,
          delGroupType:'',
          queryGroupSearchText:'',
          defaultexpandedKeys:[],
          defaultCheckedKeys:[]
        };
      },
      computed: {
        groupWritable() {
          var vm = this,
              write = true;

          if(vm.rowDataGroup && vm.rowDataGroup.write != '1') {
              write = false;
          }

          return write;
        },
        isAdmin() {
          return is_super_user == 'true';
        },
        dialogCls() {
          var vm = this;

          return {
            'no-padding': ['info','moveType'].includes(vm.moveType)
          };
        },
        isWritable() {
				  return writableMap['CODE_GNB'] == true;
			  },
        groupWritable() {
          var vm = this,
            write = true;
          
          if(vm.rowDataGroup && vm.rowDataGroup.write != '1') {
            write = false
          }

          return write;
        }
      },
      methods: {
        // 初始化
        init(){	
          var vm = this;
          vm.queryGroupList(vm.queryGroupSearchText);
        },
        // 设备组查询
        queryGroupList(val){
          var vm =this,
              params={
                search_text:val,
                isGnb: 1
              };
          vm.queryGroupSearchText = val;
          vm.defaultexpandedKeys =[];
          vm.defaultCheckedKeys = [];
          vm.deviceUrl = '';
          axios.post('${ctx}/system/deviceGroup/getFullDeviceGroupList.action',stringify(params)).then(function(response){
            let data = response.data.rows;
            vm.groupData = data;
            if(vm.groupData.length>0){
                vm.defaultexpandedKeys.push(vm.groupData[0].id);
                vm.defaultCheckedKeys.push(vm.groupData[0].children[0].id);
                vm.params_device.group_id =vm.groupData[0].children[0].id;
                vm.rowDataGroup = vm.groupData[0].children[0];
                vm.$nextTick(function(){
                  vm.$refs.groupTree.setCurrentKey(vm.params_device.group_id);
                  vm.deviceUrl = '${ctx}/cell/cpeinfos/getEnbList.action';
                })
            }
          }).catch(function(error){})
        },
        queryDevices(text) {
          this.params_device.search_text = text;
        },
        // 基站设备数据导出
        exportDevice(){
          var vm = this,
              params={isGnb: 1},
              exportUrl = '${ctx}/system/device/enodeb/exportENBCsvFile.action';
          
          params.group_id = vm.params_device.group_id;
          params.search_text = vm.params_device.search_text;
          params.like_fields = "serial_number";
          exportByForm(exportUrl,params);
        },
        // 导入设备文件确定
        uploadDevice(){
          var vm = this;
          
          if(vm.fileParams.FileName){
            vm.$refs.upload.submit();
            
          }else{
            vm.typeFlag = true;
            vm.selectFlag = true;
          }
          
        },
        // 导出设备模板
        exportTemplate(){
          
          //判断是eNB 还是 cpe的设备模板导出
          var vm = this,
              params = {isGnb: 1},
              url = "${ctx}/system/deviceGroup/downloadImportCellTemplate.action";
             
          params.group_id = this.params_device.group_id;
          params.search_text = this.searchText;
          params.like_fields = "serial_number";
          var bool = checkParams(params)

          if(!bool) return false;
          exportByForm(url,params)
        },
        /**
        * 文件上传成功函数 
        * @param res{object}   返回信息
        * @param file{object}  文件信息
        */
        checkFile(res,file){ 
          var vm = this;
          if(res.success){
            vm.$message({
              type: 'success',
              message: '<%=rb.getString("ChengGong")%>'
            });
            setTimeout(()=>{
              vm.closeImport();
            },100)
            vm.$refs.ctableDevice.refresh();
          }else{
            vm.$message({
              type: 'error',
              message: res.msg
            });
          }
          //修改已选择文件状态  
          var fileList = vm.$refs.upload.uploadFiles;
          fileList.forEach(function(file){
            file.status = 'ready';
          })
        },
        fileChange(file,fileList){    
          var vm = this,
              typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.xls' 
                        || file.name.substr(file.name.lastIndexOf("."))  === '.xlsx' 
                        || file.name.substr(file.name.lastIndexOf("."))  === '.csv';

          vm.typeFlag = typeFlag;
          vm.selectFlag = false;
          
          if(typeFlag){
            vm.fileName = file.name;
            vm.fileParams.FileName = file.name
            vm.fileParams.group_id = vm.params_device.group_id;
          }else {
            vm.fileName = '';
          }
        },
        // 选择文件
        fileSelect(){  
          var vm =this;
          vm.$refs.upload.clearFiles();
          vm.$refs['file_up'].click();
        },
        /**
        * 基站设备列表 点击更多操作出现菜单
        * @param row{object}   行数据
        * @param ev{object}   event数据
        */ 
        optDeviceClick(row,ev){ // 操作项： 1.移动到设备组 2.删除 
            var vm = this,
                writable = vm.rowDataGroup.write == '1'; 
            
            vm.rowDataDevice = row;

            vm.menusDevices= [
                  {label:'<%=rb.getString("YiDongDaoSheBeiZu")%>',cls:"el-icon el-icon-moveGroup CODE_GNB hidden",code:'move',disable: !writable},
                  //{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:'modifyDevice',show: !vm.sasEnable},
                  {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_GNB hidden",code:'deleteDevice',disable: !writable}
              ];
            
            vm.$nextTick(function(){
              document.body.click();
              vm.$refs.menuDevices.show(ev);
            });
        },
        /**
        * 基站设备列表 菜单点击事件
        * @param ev{object}   行数据
        */ 
        clickDeviceMenu(ev){
          var vm = this;
          var codes = {
                move: vm.moveToGroup,
                modifyDevice: vm.modifyDevice,
                deleteDevice: vm.deleteDevice
              };

          if(codes[ev.code]) codes[ev.code](vm.rowDataDevice);
        },
        /**
        * 移动到设备组
        * @param code{object}   设备数据
        */ 
        moveToGroup(code){
          var vm = this,
              id = '';

          if(vm.rowDataGroup) {
            id = vm.params_device.group_id;
          }

          vm.moveDlShow = true;
          vm.moveType = 'single';
          vm.moveGroupUrl = '${ctx}/system/deviceGroup/getDeviceGroupList.action?no_group_id=' + id + '&type=1&rd=' + Math.random();
        },
        // 移动到设备组 批量操作
        movecells(){
          var vm = this,
              id = '';

          if(vm.rowDataGroup) {
            id = vm.params_device.group_id;
          }
          
          vm.moveDlShow = true;
          vm.moveType = 'batch';
          vm.moveGroupUrl = '${ctx}/system/deviceGroup/getDeviceGroupList.action?no_group_id=' + id;
        },
        moveDevicesSend() {
          var vm = this ,
              ids = vm.rowDataDevice.small_cell_code + '_' + vm.rowDataDevice.product ,
              url = "${ctx}/system/deviceGroup/moveCellToDeviceGroup.action",
              idsStr = [],
              params = {
                toGroupId: vm.groupId,
                ids: ids,
                isGnb: 1
              };
              
          if(vm.moveType == 'batch') {
              idsStr = vm.ckRows.map((item,index) => {
                return item.small_cell_code + '_' + item.product;
              });

              params.ids = idsStr.join(','); 
          }

          if ( vm.groupId ){
            axios.post(url,stringify(params)).then(function(response){
              var data = response.data;
              if(data.success){
                vm.$message({
                  message: data.message || '<%=rb.getString("ChengGong")%>',
                  type: 'success',
                });	
                vm.$refs.ctableDevice.refresh();
                vm.$refs.ctableDevice.clearSelection();
                vm.ckRows = [];
                vm.closeMoveDl();
              }else{
                vm.$message({
                  type: 'error',
                  message: data.message,
                })
              }
              
            }).catch(function(error){}) 
            
          }else{
            vm.$message({
              type:'error',
              message:'Please select a bill',
            })
          }
        },
        closeMoveDl() {
          var vm = this;

          vm.$refs.moveTable.clearSelection();
          vm.moveDlShow = false;
        },
        /**
        * 基站设备修改页面
        * @param code{object}   设备数据
        */ 
        modifyDevice(row) {
          var vm = this,
              cell_code = '';
          
          cell_code = row.small_cell_code;
          vm.deviceName = row.serial_number;
          vm.showModifyDevice = true;
          
          if(row) {
            Object.assign(vm.deviceForm,{
              cell_code: cell_code,
              longitude: row.longitude,
              latitude: row.latitude,
              height: row.height,
              distance: row.distance
            })
          }else {
            Object.assign(vm.deviceForm,{
              cell_code: '',
              longitude: '',
              latitude: '',
              height: '',
              distance: ''
            })
          }
        },
        // 基站设备修改确定
        saveModifyDevice(){
          var vm = this,
            url = '${ctx}/cell/topo/setLocationInfo.action';

          vm.$refs.modifyForm.validate(function(r){
            if(r) {
              axios.post(url, stringify(vm.deviceForm)).then(function(res){
                if(res.data.success){
                  vm.$message({
                    message: '<%=rb.getString("ChengGong")%>',
                    type:'success',
                  });
                  vm.showModifyDevice = false;
                  vm.$refs.ctableDevice.refresh();
                }else {
                  vm.$message({
                    message: res.data.message,
                    type:'error',
                  });
                }
              });
            }
          })
        },
        /**
        * 基站设备删除
        * @param code{object}   设备数据
        */ 
        deleteDevice(code){
          var vm = this,
              params = {isGnb: 1},
              url = "";
              
          url = "${ctx}/system/deviceGroup/delCellinfo.action"
          params.ids = code.small_cell_code + '_' + code.product + ',';
          
          vm.$confirm('<%=rb.getString("ShanChuSheBeiHeShuJu")%>','<%=rb.getString("QueRen")%>',{
            confirmButtonText:'<%=rb.getString("QueDing")%>',
            cancalButtonText:'<%=rb.getString("QuXiao")%>',
            type:'warning'
          }).then(()=>{
            axios.post(url,stringify(params)).then(function(response){
              let data = response.data;
              if ( data.success ){
                vm.$message({
                  type: 'success',
                  message: '<%=rb.getString("ChengGong")%>'
                });
                vm.$refs.ctableDevice.refresh();
                vm.$refs.ctableDevice.clearSelection();
              }else {
                vm.$message.error(data.message)
              }
              
            }).catch(function(error){})
            
          }).catch(()=>{})
        },
        // 基站设备 批量删除
        deleteCells(){
          var vm = this,
              params = {isGnb: 1},
              url = "${ctx}/system/deviceGroup/delCellinfo.action",
              idsStr = [];

          idsStr = vm.ckRows.map((item,index) => {
            return item.small_cell_code + '_' + item.product;
          })
          params.ids = idsStr.join(',');
          
          vm.$confirm('<%=rb.getString("ShanChuSheBeiHeShuJu")%>','<%=rb.getString("QueRen")%>',{
            confirmButtonText:'<%=rb.getString("QueDing")%>',
            cancalButtonText:'<%=rb.getString("QuXiao")%>',
            type:'warning'
          }).then(()=>{
            axios.post(url,stringify(params)).then(function(response){
              let data = response.data;
              if ( data.success ){
          
                vm.$message.success('<%=rb.getString("ChengGong")%>')
                vm.$refs.ctableDevice.refresh();
                vm.ckRows = [];
                vm.$refs.ctableDevice.clearSelection();
              }else {
                vm.$message.error(data.message)
              }
              
            }).catch(function(error){})
            
          }).catch(()=>{
            
          })
        },
        // 添加一级 设备组
        addDeviceGroup(){
          var vm = this;
          vm.showDeviceGroupDialog = true;
          vm.viewDeviceGroupDialog = false;
          vm.dialogTitle = '<%=rb.getString("TianJia")%>';
          vm.deviceGroupDialogWidth = '500px';
          vm.stairGroupType = 'add'
        },
        // 一级 设备组 新增/修改提交 
        addDeviceGroupSubmit(){
          var param={}  , vm = this , url;
            param.groupName = vm.groupForm.groupName;
            if (vm.stairGroupType == "add") {
              url = "${ctx}/system/deviceGroup/addTopDevice.action";
            } else {
              url = "${ctx}/system/deviceGroup/modTopDevice.action";
              param.groupId = vm.rowDataGroup.id;
            }
              vm.$refs.deviceGroupDialogForm.validate((valid) => {
                if(valid){
                  axios.post(url,stringify(param)).then(function(response){
                    let data = response.data;
                    if ( data.success ){
                      vm.$message({
                      message: '<%=rb.getString("ChengGong")%>',
                      type:'success',
                    });	
                    }else {
                      vm.$message.error(data.message)
                    }
                    vm.closeDeviceGroupDialog();

                  }).catch(function(error){})
                }else{
                  return false;
                }
              })
        },
        // 关闭 一级 设备组弹窗
        closeDeviceGroupDialog(){
          var vm = this,
              params={
                groupName:'',
              };
          vm.$refs.deviceGroupDialogForm.resetFields();
          Object.assign(vm.groupForm,params);
          vm.showDeviceGroupDialog = false;
          vm.queryGroupList(vm.queryGroupSearchText);
        },
        // 二级设备组 行点击事件
        groupRowClick(data,node,ev){
          var vm = this;
          if(!data.children){
              vm.rowDataGroup = data;
              vm.params_device.group_id = data.id;
              vm.$nextTick(function(){
                vm.deviceUrl = '${ctx}/cell/cpeinfos/getEnbList.action';
                vm.$refs.ctableDevice.clearSelection();
              })
          }
        },
        /**
        * 点击设备组操作 生成下拉选项
        * @param row:当前点击项数据 
        * 操作项： 1.信息  2.修改  3.删除  
        */
        groupOpClick(node,data,ev){
            var vm = this,addShowFlag = false,editDisFlag=false;
          
            vm.rowDataGroup = data;
            if(data.children){
              addShowFlag = true
            }else{
              addShowFlag = false
            }
            if(data.built_in == '1' || (data.write != undefined && data.write != '1')){
              editDisFlag = true;
            }
            vm.menusGroup= [
              {label:'Add Subgroup',cls:"el-icon el-icon-operation-add",code:'add',show:addShowFlag},
              {label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info',show:!addShowFlag},
              {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:'edit',disable:editDisFlag},
              {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del',disable:editDisFlag}
            ]
            vm.$nextTick(function(){
              document.body.click();
              vm.$refs.menuGroup.show(ev);
            });
            event.stopPropagation();
        },
        /**
        * 设备组更多操作栏 单点方法 
        * @param ev:当前点项
        */
        clickMenu(ev){ 
            var vm = this;
            var codes = {
                add:vm.addSubgroup,
                info:vm.viewSubGroupInfo,
                edit:vm.modifyGroup,
                del:vm.deleleGroup
            }
            if(codes[ev.code]){
              codes[ev.code](vm.rowDataGroup.id)
            }
        },
        // 添加二级 设备组
        addSubgroup(){
            var vm = this;
            vm.showWindowInfo = true;
            vm.dialogTitle = '<%=rb.getString("TianJia")%>';
            vm.windowWidth = '1000px';
            vm.dialogUrl = '${ctx}/gnb/device/toGroupAddPage.action';
            vm.moveType = 'addGroup';
            vm.curGroup = vm.rowDataGroup;
        },
        /**
        * 查看设备组详情
        * @param id 传入当前数据的id	
        */
        viewSubGroupInfo(id){
            var vm = this;
          
            vm.curGroup = vm.rowDataGroup;
            vm.showWindowInfo = true;
            vm.dialogTitle = '<%=rb.getString("XinXi")%>';
            vm.windowWidth = '1000px';
            vm.dialogUrl = '${ctx}/gnb/device/toGroupAddPage.action';
            vm.moveType = 'info';
            
        },
        /**
        * 修改设备组
        * @param id 传入当前数据的id	
        */
        modifyGroup(id){
            var vm = this,
              params={
                groupName:vm.rowDataGroup.group_name,
              };
            if(vm.rowDataGroup.children){
                vm.showDeviceGroupDialog = true;
                vm.viewDeviceGroupDialog = false;
                vm.dialogTitle = '<%=rb.getString("XiuGai")%>';
                vm.deviceGroupDialogWidth = '500px';
                vm.stairGroupType = 'modify';
                Object.assign(vm.groupForm,params);
            }else{
                vm.curGroup = vm.rowDataGroup;
                vm.showWindowInfo = true;
                vm.dialogTitle = '<%=rb.getString("XiuGai")%>';
                vm.windowWidth = '1000px';
                vm.dialogUrl = '${ctx}/gnb/device/toGroupAddPage.action';
                vm.moveType = 'modify';
            }
          
        },
        /**
        * 删除设备组
        * @param id 传入当前数据的id	
        */
        deleleGroup(id){
          var vm = this;
          if(vm.rowDataGroup.children){
            vm.delGroupType = 'stair';
          }else{
            vm.delGroupType = 'second';
          }
          this.showDeviceGroupInfo = true;

        },
        //删除设备组确认
        deleteDeviceGroup(){
          var vm = this,
              urls="",
              params = {};
            if(vm.delGroupType == 'stair'){
              params.groupId = vm.rowDataGroup.id;
              urls = '${ctx}/system/deviceGroup/delTopDevice.action'
            }else{
              params.id = vm.rowDataGroup.id;
              urls = "${ctx}/system/deviceGroup/deleteDeviceGroup.action"
            }
            axios.post(urls,stringify(params)).then(function(response){
              let data = response.data;
              if ( data.success ){
                vm.$message.success(data.message);
                vm.showDeviceGroupInfo = false;
                vm.queryGroupList(vm.queryGroupSearchText);
              }else {
                vm.$message.error(data.message)
              }
              
            }).catch(function(error){})
        },
        // 弹窗打开成功传递参数  
        openDialogSuc(){
          var vm = this;
          if(vm.moveType == 'single'){
            eventBus.$emit('open-dialog');
          }else if(vm.moveType == 'batch') {
            eventBus.$emit('open-dialog-moveto');
          }else if(vm.moveType == 'addGnb'){
            var groupId = vm.rowDataGroup? vm.rowDataGroup.id:'';
            eventBus.$emit('init-gnb',{groupId: groupId});
          }else {
            eventBus.$emit('init-group',{type: vm.moveType, groupId: vm.curGroup.id});
          }
        },
        // 关闭弹窗
        closeDialog(){
          var vm = this;
          
          vm.showWindowInfo = false;
          vm.dialogUrl = '';
        },
      
        closeImport() {
          var vm = this;

          vm.fileName = '';
          vm.fileParams.FileName = '';
          vm.$refs.upload.clearFiles();
          document.body.click();
        },
      
        handerClose() {
          this.$refs.menuGroup.hide();
          this.$refs.menuDevices.hide();
        },
        change(s) {
          var vm = this;

          vm.ckRows = s;
        },
        selectGroup(currentRow,oldCurrentRow){
          var vm = this;
          vm.rowDataGroup = currentRow;
          currentRow && (vm.params_device.group_id = currentRow.id);
          vm.$nextTick(function(){
            vm.deviceUrl = '${ctx}/cell/cpeinfos/getEnbList.action';
            vm.$refs.ctableDevice.clearSelection();
          })
        },
        addGroup() {// 添加设备组
          var vm = this;

          vm.showWindowInfo = true;
          vm.dialogTitle = '<%=rb.getString("TianJia")%>';
          vm.windowWidth = '1000px';
          vm.dialogUrl = '${ctx}/gnb/device/toGroupAddPage.action';
          vm.moveType = 'addGroup';
          vm.curGroup = '';
        },
        addGnb() {// 条件gnb设备
          var vm = this,
              groupId = vm.rowDataGroup? vm.rowDataGroup.id:'';
          vm.showWindowInfo = true;
          vm.dialogTitle = '<%=rb.getString("TianJia")%>';
          vm.windowWidth = '500px';
          vm.dialogUrl = '${ctx}/gnb/device/toGnodeBAddPage.action';
          vm.moveType = 'addGnb';
        },
        groupLoadSuccess(data) {
        	if(data && data.rows && data.rows.length) this.$refs.ctableGroup.setCurrentRow(data.rows[0]);
        },
        // 设备表格刷新
        refreshDeviceTable(){
          var vm = this;
          vm.$refs.ctableDevice.refresh();
        }
      },
      mounted() {
          var vm = this;
          vm.init();
          eventBus.$off('close-gnb-dialog').$on('close-gnb-dialog', this.closeDialog);
          eventBus.$off('refresh-group-list').$on('refresh-group-list', this.queryGroupList);
          eventBus.$off('refresh-device-list').$on('refresh-device-list', this.refreshDeviceTable);
      }
    })
  </script>
</body>
</html>